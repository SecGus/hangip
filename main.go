package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type CIDRMapping struct {
	IPPrefix          string `json:"ip_prefix"`
	Region            string `json:"region"`
	Service           string `json:"service"`
	NetworkBorderGroup string `json:"network_border_group"`
}

type CIDRList struct {
	CIDRs []CIDRMapping `json:"cidrs"`
}

type SubdomainInfo struct {
	Subdomain string `json:"subdomain"`
	IP        string `json:"ip"`
	Region    string `json:"region"`
	Service   string `json:"service"`
}

func main() {
	// Command-line flags
	ipOnly := flag.Bool("iponly", true, "Output only the IP address")
	verbose := flag.Bool("v", false, "Output IP, region, and subdomain")
	jsonOutput := flag.Bool("json", false, "Output in JSON format")
	cidrFile := flag.String("c", "cidr_mappings.json", "Path to the CIDR mapping file")
	flag.Parse()

	// Read the JSON file containing CIDR mappings
	jsonFile, err := os.Open(*cidrFile)
	if err != nil {
		fmt.Println("Error opening JSON file:", err)
		return
	}
	defer jsonFile.Close()

	// Read the JSON content
	byteValue, _ := ioutil.ReadAll(jsonFile)

	// Parse the JSON content into a CIDRList struct
	var cidrList CIDRList
	err = json.Unmarshal(byteValue, &cidrList)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	// Read subdomains from standard input
	subdomains, err := readSubdomains()
	if err != nil {
		fmt.Println("Error reading subdomains:", err)
		return
	}

	// Process each subdomain concurrently
	subdomainInfo := make(chan SubdomainInfo)
	var wg sync.WaitGroup
	wg.Add(len(subdomains))

	for _, subdomain := range subdomains {
		go func(subdomain string) {
			defer wg.Done()

			ipAddresses, err := getIPAddresses(subdomain)
			if err != nil {
				return
			}

			if len(ipAddresses) == 0 {
				// Skip subdomain without associated IP addresses
				return
			}

			for _, ip := range ipAddresses {
				for _, cidr := range cidrList.CIDRs {
					_, ipNet, err := net.ParseCIDR(cidr.IPPrefix)
					if err != nil {
						continue
					}

					if ipNet.Contains(ip) {
						info := SubdomainInfo{
							Subdomain: subdomain,
							IP:        ip.String(),
							Region:    cidr.Region,
							Service:   cidr.Service,
						}

						subdomainInfo <- info
						break
					}
				}
			}
		}(subdomain)
	}

	go func() {
		wg.Wait()
		close(subdomainInfo)
	}()

	// Print the results
	if *jsonOutput {
		printJSONResults(subdomainInfo)
	} else {
		printResults(subdomainInfo, *ipOnly, *verbose)
	}
}

func readSubdomains() ([]string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	subdomains := []string{}

	for scanner.Scan() {
		subdomain := strings.TrimSpace(scanner.Text())
		if subdomain != "" {
			subdomains = append(subdomains, subdomain)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return subdomains, nil
}

func getIPAddresses(domain string) ([]net.IP, error) {
	cmd := exec.Command("nslookup", domain)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	ipAddresses := []net.IP{}
	for _, line := range lines {
		if strings.HasPrefix(line, "Address:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				ip := net.ParseIP(fields[1])
				if ip != nil {
					ipAddresses = append(ipAddresses, ip)
				}
			}
		}
	}

	return ipAddresses, nil
}

func printJSONResults(results chan SubdomainInfo) {
	infoList := []SubdomainInfo{}
	for info := range results {
		infoList = append(infoList, info)
	}

	jsonBytes, err := json.MarshalIndent(infoList, "", "    ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	fmt.Println(string(jsonBytes))
}

func printResults(results chan SubdomainInfo, ipOnly bool, verbose bool) {
	for info := range results {
		if verbose {
			fmt.Println("Subdomain:", info.Subdomain)
			fmt.Println("IP:", info.IP)
			fmt.Println("Region:", info.Region)
			fmt.Println("Service:", info.Service)
			fmt.Println()
		} else if ipOnly {
			fmt.Println(info.IP)
		} else {
			fmt.Printf("%s\t%s\t%s\n", info.Subdomain, info.IP, info.Region)
		}
	}
}
