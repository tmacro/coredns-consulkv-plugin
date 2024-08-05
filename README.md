# CoreDNS ConsulKV Plugin

This plugin enables CoreDNS to use Consul's Key-Value store as a backend for DNS records. It supports both forward and reverse DNS lookups, as well as wildcard entries.

## Features

- Use Consul KV as a DNS record store
- Support for A and PTR records (extensible to other types)
- Wildcard domain support
- Configurable TTL with default value
- Metrics for monitoring (compatible with Prometheus)

## Installation

To use this plugin, you need to compile it into CoreDNS. Add the following line to the `plugin.cfg` file in your CoreDNS source code:

```
consulkv:github.com/mwantia/coredns-consulkv-plugin
```

Then, rebuild CoreDNS with:

```sh
go get github.com/mwantia/coredns-consulkv-plugin
go generate
go build
```

## Configuration

Add the plugin to your CoreDNS configuration file (Corefile):

```corefile
. {
    consulkv {
        address http://127.0.0.1:8500
        prefix dns
        token <consul-acl-token>
        zones example.com 100.in-addr.arpa
        fallthrough
    }
}
```

### Configuration Options

- `address`: Consul HTTP address (default: `http://localhost:8500`)
- `prefix`: Key prefix in Consul KV (default: `dns`)
- `token`: Consul ACL token (optional)
- `zones`: DNS zone to be handled by this plugin (can be specified multiple times)
- `fallthrough`: If set, passes the request to the next plugin when no record is found (optional, default: false)

## Consul KV Structure

DNS records are stored in Consul's KV store with the following structure:

```
<prefix>/<zone>/<record>
```

For example:

- `dns/example.com/www`
- `dns/100.in-addr.arpa/86.203.96`

The value for each key should be a JSON object with the following structure:

```json
{
  "ttl": 3600,
  "records": [
    {
      "type": "<type>",
      "value": "<see-examples-for-details>"
    }
  ]
}
```

### Special Entries

- Zone apex (root domain): Use `@` as the record name.
- Wildcard: Use `*` as the record name.

## Examples

1. A record for www.example.com:

   Key: `dns/zones/example.com/www`
   Value:
   ```json
   {
     "ttl": 3600,
     "type": "A",
     "records": [
       {
         "type": "A",
         "value": ["192.168.1.10"]
       }
     ]
   }
   ```

2. PTR record for reverse DNS:

   Key: `dns/zones/100.in-addr.arpa/86.203.96`
   Value:
   ```json
   {
     "ttl": 3600,
     "records": [
       {
         "type": "PTR",
         "value": ["www.example.com"]
       }
     ]
   }
   ```

3. Wildcard record for *.example.com with additional TXT:

   Key: `dns/zones/example.com/*`
   Value:
   ```json
   {
     "ttl": 3600,,
     "records": [
       {
         "type": "A",
         "value": ["192.168.1.100"]
       },
       {
          "type": "TXT",
          "value": ["This is some additional information"]
       }
     ]
   }
   ```

4. SRV record for a service:

   Key: `dns/zones/example.com/_sip._tcp`
   Value:
   ```json
   {
     "ttl": 3600,
     "records": [
       {
         "type": "SRV",
         "value": [
         {
           "target": "sip.example.com",
           "port": 5060,
           "priority": 10,
           "weight": 100
         }
        ]
       }
     ]
   }

3. CNAME record for test.example.com:

   Key: `dns/zones/example.com/test`
   Value:
   ```json
   {
     "ttl": 3600,
     "records": [
       {
         "type": "CNAME",
         "value": "www.example.com"
       }
     ]
   }
   ```

## Metrics

This plugin exposes the following metrics for Prometheus:

- `coredns_consulkv_successful_queries_total{zone, qtype}`: Count of successful DNS queries
- `coredns_consulkv_consul_errors_total`: Count of Consul connection/query errors
- `coredns_consulkv_invalid_responses_total`: Count of invalid DNS responses generated

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
