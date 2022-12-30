# SQM

SQM is a go program that sets up using Linux qdisc mechanism using the CAKE integrated scheduler. See http://www.bufferbloat.net/projects/cerowrt/wiki/Smart_Queue_Management for more information

It is based upon the existing scripts from github.com/tohojo/sqm-scripts - which has a lot more information - but has the following changes:

* Runs as a daemon
* Checks on timed intervals to make sure everything is correct
* Tears down as much as possible on shutdown

## Use cases

The author uses this at home in his home-baked Fedora-based VDSL router, that runs a PPP session via a Zyxel modem bridge.

## Running

Run `go run cmd/sqm` or `sqm` after building

### Options

```
  -e, ----egress-oid string    SNMP OID for egress (default "1.3.6.1.2.1.10.97.1.1.2.1.10.2")
  -i, ----ingress-oid string   SNMP OID for ingress (default "1.3.6.1.2.1.10.97.1.1.2.1.10.1")
  -l, ----snmp-host string     SNMP Host (default "192.168.2.1")
  -h, --help                   help for sqm
  -d, --interface string       Device to configure (default "ppp0")
```

## Building

Run `mage install`
