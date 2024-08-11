# CoreDNS ConsulKV Plugin

This plugin enables CoreDNS to use Consul's Key-Value store as a backend for DNS records. \
It supports both forward and reverse DNS lookups, as well as wildcard entries.

## Features

- Use Consul KV as a DNS record store
- Support for forward and reverse DNS records
- Wildcard and root domain support
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
        token ""
        zones example.com 0.168.192.in-addr.arpa
    }
}
```

### Configuration Options

- `address`: Consul HTTP address (default: `http://localhost:8500`)
- `prefix`: Key prefix in Consul KV (default: `dns`)
- `token`: Consul ACL token (optional)
- `zones`: DNS zone to be handled by this plugin (can be specified multiple times)
- `no_cache`: Disables the internal cache of the Consul client
- `no_flattening`: Disables CNAME flattening so only the CNAME record will be returned

Just creating a zone prefix in Consul KV is not enough. \
This plugin requires that all zones that should be handled to be defined via `zones`.

## Consul KV Structure

DNS records are stored in Consul's KV store with the following structure:

```
<prefix>/<zone>/<record>
```

For example:

- `dns/example.com/www`
- `dns/0.168.192.in-addr.arpa/1`

The value for each key must be a JSON object with the following structure:

```json
{
  "ttl": 3600,
  "records": [
    {
      "type": "<query-type>",
      "value": "<see-below-for-examples>"
    }
  ]
}
```

### Special Entries

- Zone apex (root domain): Use `@` as the record name.
- Wildcard: Use `*` as the record name.

## Examples

1. SOA root record with NS for example.com

   Key: `dns/zones/example.com/@`

   ```json
   {
     "ttl": 3600,
     "records": [
       {
         "type": "SOA",
         "value": {
           "mname": "ns.example.com",
           "rname": "hostmaster.example.com",
           "serial": 2024081001,
           "refresh": 7200,
           "retry": 3600,
           "expire": 1209600,
           "minimum": 3600
         },
         {
           "type": "NS",
           "value": ["ns.example.com"]
         }
       }
     ]
   }
   ```

2. A record for ns.example.com:

   Key: `dns/zones/example.com/ns`
   Value:
   ```json
   {
     "ttl": 3600,
     "records": [
       {
         "type": "A",
         "value": ["192.168.0.5"]
       }
     ]
   }
   ```

3. PTR record for reverse DNS:

   Key: `dns/zones/0.168.192.in-addr.arpa/5`
   Value:
   ```json
   {
     "ttl": 3600,
     "records": [
       {
         "type": "PTR",
         "value": ["ns.example.com"]
       }
     ]
   }
   ```

4. Wildcard record for `*.example.com` with additional TXT:

   Key: `dns/zones/example.com/*`
   Value:
   ```json
   {
     "ttl": 3600,
     "records": [
       {
         "type": "A",
         "value": ["192.168.0.100"]
       },
       {
          "type": "TXT",
          "value": ["Additional information displayed as TXT"]
       }
     ]
   }
   ```

5. SRV record for a service:

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

6. CNAME record for test.example.com:

   Key: `dns/zones/example.com/www`
   Value:
   ```json
   {
     "ttl": 3600,
     "records": [
       {
         "type": "CNAME",
         "value": "example.com"
       }
     ]
   }
   ```

6. HTTPS for service.example.com with additional A records:

   Key: `dns/zones/example.com/service`
   Value:
   ```json
   {
     "ttl": 3600,
     "records": [
       {
         "type": "A",
         "value": ["192.168.0.10", "192.168.0.11"]
       },
       {
         "type": "HTTPS",
         "value": [
           {
             "priority": 1,
             "target": "docker1.example.com",
             "params": {
               "alpn": "h2,http/1.1",
               "port": "443",
               "ipv4hint": "192.168.0.10"
             }
           },
           {
             "priority": 1,
             "target": "docker2.example.com",
             "params": {
               "alpn": "h2,http/1.1",
               "port": "443",
               "ipv4hint": "192.168.0.11"
             }
           }
         ]
       }
     ]
   }
   ```

## Metrics

This plugin exposes the following metrics for Prometheus:

* `coredns_consulkv_plugin_errors_total{error}`: 
  * Count the amount of errors within the plugin \
    The list of possible errors are: 
    * `CONSUL_GET`: Occures when ConsulKV was unable to connect to Consul
    * `SOA_GET`: Occures when ConsulKV was unable to load any SOA entries from Consul or as default
    * `WRITE_MSG`: Occures when ConsulKV was unable to write the response to CoreDNS due to an internal panic
    * `JSON_UNMARSHAL`: Occures when ConsulKV was unable to unmarshal the received json value from Consul
* `coredns_consulkv_consul_request_duration_seconds{status, le}`
  * Histogram of the time (in seconds) each request to Consul took \
    The list of possible statuses are:
    * `NOERROR`: Occures when ConsulKV was successfully able to receive data from Consul
    * `NODATA`: Occures when ConsulKV was able to receive a response from Consul but no record exists
    * `ERROR`: Occures when ConsulKV was unable to connect to Consul
* `coredns_consulkv_query_requests_total{zone, type}`
  * Count the amount of queries received as request by the plugin \
    The label `zone` defines the zonename requested in this query (Example: `example.com.`) \
    The label `type` defines the query type that was requested (Example: `A`, `CNAME`)
* `coredns_consulkv_query_responses_successful_total{zone, type}`
  * Count the amount of successful queries handled and responded to by the plugin \
    The label `zone` defines the zonename requested in this query (Example: `example.com.`) \
    The label `type` defines the query type that was requested (Example: `A`, `CNAME`)
* `coredns_consulkv_query_responses_failed_total{zone, type, error}`
  * Count the amount of failed queries handled by the plugin \
    The label `zone` defines the zonename requested in this query (Example: `example.com.`) \
    The label `type` defines the query type that was requested (Example: `A`, `CNAME`) \
    The list of possible errors are: 
    * `ERROR`: Occures when ConsulKV wasn't able to complete the request due to internal errors
    * `NODATA`: Occures when ConsulKV was unable to find a record matching the request
    * `NXDOMAIn`: Occures when ConsulKV was unable to find a record and was unable to return any form of data, like `SOA`

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.
