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
  -json
        Output in JSON format
  -v    Output IP, region, and subdomain
```

## Installation
```
git clone https://github.com/SecGus/hangip.git
cd hangip
go build main.go
mv ./main /usr/bin/hangip
```
