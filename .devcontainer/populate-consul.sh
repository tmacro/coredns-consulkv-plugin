#!/bin/bash

# Wait for Consul to be ready
until consul members; do
    echo "Waiting for Consul to start..."
    sleep 1
done

# Function to put key-value pair in Consul
put_kv() {
    consul kv put "$1" "$2"
}

# Populate Consul KV with example DNS data
put_kv "dns/config" '{
  "zones": ["example.com", "0.168.192.in-addr.arpa"],
  "flattening": "local"
}'

put_kv "dns/zones/example.com/@" '{
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
      }
    },
    {
      "type": "NS",
      "value": ["ns.example.com"]
    },
    {
      "type": "A",
      "value": ["192.168.0.2"]
    }
  ]
}'

put_kv "dns/zones/example.com/www" '{
  "ttl": 3600,
  "records": [
    {
      "type": "A",
      "value": ["192.168.0.3"]
    }
  ]
}'

put_kv "dns/zones/example.com/alias" '{
  "ttl": 3600,
  "records": [
    {
      "type": "CNAME",
      "value": "www.example.com"
    }
  ]
}'

put_kv "dns/zones/example.com/txt" '{
  "ttl": 3600,
  "records": [
    {
      "type": "TXT",
      "value": ["This is a test"]
    }
  ]
}'

put_kv "dns/zones/0.168.192.in-addr.arpa/2" '{
  "ttl": 3600,
  "records": [
    {
      "type": "PTR",
      "value": ["example.com"]
    }
  ]
}'

put_kv "dns/zones/0.168.192.in-addr.arpa/3" '{
  "ttl": 3600,
  "records": [
    {
      "type": "PTR",
      "value": ["www.example.com"]
    }
  ]
}'

echo "Consul KV populated with example DNS data."