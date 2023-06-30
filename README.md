# hangip
Takes in a JSON file of CIDRs and matches subdomains to their IPs

Note that `cidr_mappings.json` should be a JSON file including the CIDR mappings including IP range, region, and company, such as the following format:
```
{"cidrs":[{
      "ip_prefix": "3.2.34.0/26",
      "region": "af-south-1",
      "service": "AMAZON",
      "network_border_group": "af-south-1"
    },
    {
      "ip_prefix": "3.5.140.0/22",
      "region": "ap-northeast-2",
      "service": "AMAZON",
      "network_border_group": "ap-northeast-2"
    },
    {
      "ip_prefix": "13.34.37.64/27",
      "region": "ap-southeast-4",
      "service": "AMAZON",
      "network_border_group": "ap-southeast-4"
    }
]}
```

## Usage
```
Usage of hangip:
  -c string
        Path to the CIDR mapping file (default "cidr_mappings.json")
  -dbbp string
        BBP for database
  -dbd string
        Database name
  -dbip string
        Database IP
  -dbp string
        Database password
  -dbport string
        Database port (default "3306")
  -dbu string
        Database username
  -json
        Output in JSON format
  -v    Output IP, region, and subdomain
```

## Installation
Note installation does not include instructions to DB setup.
```
git clone https://github.com/SecGus/hangip.git
cd hangip
go build main.go
mv ./main /usr/bin/hangip
```

## Database Setup

This tool can be used to automatically add potential IPs for taking over to a MySQL database. Note that the default for all database arguments is "" except for port, which is 3306. The tool uses a combination of IP:subdomain base64 encoded as a key.
```mysql
CREATE DATABASE hangip;
USE hangip;
CREATE TABLE ips (id VARCHAR(255) PRIMARY KEY, ip VARCHAR(255), subdomain VARCHAR(255), region VARCHAR(255), service VARCHAR(255), bbp VARCHAR(255));
```
Then run the tool with it's required arguments.
```
cat subdomains | hangip -dbbp hackerone -dbd hangip -dbip 127.0.0.1 -dbp 'password' -dbu admin
```
