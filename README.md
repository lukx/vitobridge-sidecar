# vitobridge-sidecar

**DO NOT USE!** Investigation repository, this is not working yet!

The goal is to translate Viessmann Vitoconnect EEBUS signals to a more
widespread protocol (probably MQTT).

## Investigation
Have a look at the [Findings from the Vitoconnect Discovery](./doc/investigation/discovery.md)

## Setup
There are unmerged changes in the Branch "add-hvac-features" of https://github.com/lukx/eebus-go/tree/add-hvac-features

Checkout my version of eebus-go next to this repository; go.mod points to that folder using "replace"

## Roadmap
* [x] Connect & Pair to Vitoconnect
* [x] Read (Poll) Masurement Data & Description
    * [ ] Note: My device started to respond error measurements for now, why?
* [x] Read (Poll) HVAC Smart Grid Overrun
* [ ] Subscribe to Measurement & HVAC Smart Grid Overrun
* [ ] Bind HVAC Function using the Binding feature
* [ ] Write HVAC Overrun

Next Level:
* [ ] Supply current PV Overload as a Measurement on _this_ side, allowing the VitoControll PV Usage to kick in. 
  
## Usage

```sh
go run cmd/main.go <serverport> <remoteski> <certfile> <keyfile>
# e.g.
go run cmd/main.go 4712 d253d08ace1dcfa2bbef88260751e8090cbfc568 ./keys/hems.crt ./keys/hems.key 

```

Example certificate and key files are located in the keys folder

### Explanation

The remoteski is from the eebus service to connect to.
If no certfile or keyfile are provided, they are generated and printed in the console so they can be saved in a file and later used again. The local SKI is also printed.

# Attributions

* [DerAndereAndi](https://github.com/DerAndereAndi) and the [EVCC Project](https://evcc.io/) for making eebus readable
  to the average developer.
