package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/savioxavier/termlink"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"resty.dev/v3"
)

const K3S_URL = "https://update.k3s.io/v1-release/channels"

type K3SChannelNames []string

type K3SLinks struct {
	Self string `json:"self"`
}

type K3SChannel struct {
	Type          string   `json:"type"`
	Id            string   `json:"id"`
	Links         K3SLinks `json:"links"`
	Name          string   `json:"name"`
	Latest        string   `json:"latest"`
	LatestRegexp  *string  `json:"latestRegexp,omitempty"`
	ExcludeRegexp *string  `json:"excludeRegexp,omitempty"`
}

type K3SCollection struct {
	Type         string         `json:"type"`
	Links        K3SLinks       `json:"links"`
	Actions      map[string]any `json:"actions"`
	ResourceType string         `json:"resourceType"`
	Data         []K3SChannel   `json:"data"`
}

func (c *K3SCollection) Channels() []string {
	names := make([]string, len(c.Data))
	for _, v := range c.Data {
		names = append(names, v.Id)
	}
	return names
}

func getCollection() (*K3SCollection, error) {
	client := resty.New()
	defer client.Close()

	res, err := client.R().Get(K3S_URL)
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve K3S channels. Error: %w", err)
	}
	var collection K3SCollection
	err = json.Unmarshal(res.Bytes(), &collection)
	if err != nil {
		return nil, fmt.Errorf("Unable to unmarshal JSON. Error: %w", err)
	}

	return &collection, nil
}

func getChannel(collection *K3SCollection, name string) *K3SChannel {
	for _, c := range collection.Data {
		if c.Id == name {
			return &c
		}
	}
	return nil
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "k3s-cryer [channel]",
		Short: "Find the latest version of a K3S release channel",
		Args:  cobra.MaximumNArgs(1),
		Run: func(_ *cobra.Command, args []string) {
			var name string
			if len(args) == 0 {
				name = "stable"
			} else {
				name = args[0]
			}
			ok := PrintRelease(name)
			if !ok {
				os.Exit(1)
			}
		},
	}
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, color.RedString("Unable to run for some reason: %s"), err)
	}
}

func PrintRelease(name string) bool {
	collection, err := getCollection()
	if err != nil {
		fmt.Fprint(os.Stderr, color.RedString(err.Error()))
	}
	channel := getChannel(collection, name)
	if channel != nil {
		fmt.Printf("The latest release of the K3S [%s] channel is: %s\n", color.BlueString(name), color.GreenString(channel.Latest))
		link := channel.Links.Self
		if term.IsTerminal(int(os.Stdout.Fd())) {
			link = termlink.Link(link, link)
		}
		fmt.Printf("Link: %s\n", color.GreenString(link))
		return true
	} else {
		fmt.Fprintf(os.Stderr, color.RedString("%s not found in channels\n"), name)
		fmt.Fprintln(os.Stderr, "Available Channels")
		for _, v := range collection.Channels() {
			fmt.Fprintf(os.Stderr, "• %s", v)
		}
		return false
	}
}
