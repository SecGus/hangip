package main

import (
        "bufio"
        "database/sql"
        "encoding/json"
        "encoding/base64"
        "flag"
        "fmt"
        "io/ioutil"
        "net"
        "os"
        "os/exec"
        "strings"
        "sync"

        _ "github.com/go-sql-driver/mysql"
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

type DBConfig struct {
        Username string
        Password string
        IP       string
        Port     string
        DBName   string
        BBP      string
}

func (dbConfig *DBConfig) IsSet() bool {
        return dbConfig.Username != "" && dbConfig.Password != "" && dbConfig.IP != "" && dbConfig.DBName != "" && dbConfig.BBP != ""
}

func main() {
        // Command-line flags
        verbose := flag.Bool("v", false, "Output IP, region, and subdomain")
        jsonOutput := flag.Bool("json", false, "Output in JSON format")
        cidrFile := flag.String("c", "cidr_mappings.json", "Path to the CIDR mapping file")

        // Database flags
        dbUsername := flag.String("dbu", "", "Database username (If any of the `db` arguments are specified, all should be specified)")
        dbPassword := flag.String("dbp", "", "Database password")
        dbIP := flag.String("dbip", "", "Database IP")
        dbPort := flag.String("dbport", "3306", "Database port")
        dbName := flag.String("dbd", "", "Database name")
        bbp := flag.String("dbbp", "", "BBP for database")
        flag.Parse()

        // Define the dbConfig
        dbConfig := DBConfig{
                Username: *dbUsername,
                Password: *dbPassword,
                IP:       *dbIP,
                Port:     *dbPort,
                DBName:   *dbName,
                BBP:      *bbp,
        }

        // Connect to the database if database information is provided
        var db *sql.DB
        var err error

        if dbConfig.IsSet() {
                db, err = connectToDB(dbConfig)
                if err != nil {
                        fmt.Println("Error connecting to database:", err)
                        return
                }
                defer db.Close()
        }

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

        // Process each subdomain
        for _, subdomain := range subdomains {
                go func(subdomain string, db *sql.DB, dbConfig DBConfig) {
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

                                                if db != nil {
                                                        if err := insertData(db, info, dbConfig.BBP); err != nil {
                                                                fmt.Println("Error inserting data:", err)
                                                        }
                                                }

                                                break
                                        }
                                }
                        }
                }(subdomain, db, dbConfig)
        }

        go func() {
                wg.Wait()
                close(subdomainInfo)
        }()

        // Print the results
        if *jsonOutput {
                printJSONResults(subdomainInfo)
        } else {
                printResults(subdomainInfo, *verbose)
        }
}

func connectToDB(dbConfig DBConfig) (*sql.DB, error) {
        conn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbConfig.Username, dbConfig.Password, dbConfig.IP, dbConfig.Port, dbConfig.DBName)
        db, err := sql.Open("mysql", conn)
        if err != nil {
                return nil, err
        }

        return db, nil
}

func insertData(db *sql.DB, info SubdomainInfo, bbp string) error {
        query := `INSERT INTO ips (id, subdomain, ip, region, service, bbp) VALUES (?, ?, ?, ?, ?, ?)`
        _, err := db.Exec(query, base64.StdEncoding.EncodeToString([]byte(info.IP+":"+info.Subdomain)), info.Subdomain, info.IP, info.Region, info.Service, bbp)
        return err
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

func printResults(results chan SubdomainInfo, verbose bool) {
        for info := range results {
                if verbose {
                        fmt.Println("Subdomain:", info.Subdomain)
                        fmt.Println("IP:", info.IP)
                        fmt.Println("Region:", info.Region)
                        fmt.Println("Service:", info.Service)
                        fmt.Println()
                } else {
                        fmt.Printf("%s\n", info.IP)
                }
        }
}
