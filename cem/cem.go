package cem

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/DerAndereAndi/eebus-go/service"
	"github.com/DerAndereAndi/eebus-go/spine/model"
)

type Cem struct {
	brand        string
	model        string
	serialNumber string
	identifier   string
	myService    *service.EEBUSService
	tmpLukas     uint
	hvacSupport  *Hvac
}

func NewCEM(brand, model, serialNumber, identifier string) *Cem {
	return &Cem{
		brand:        brand,
		model:        model,
		serialNumber: serialNumber,
		identifier:   identifier,
		tmpLukas:     0,
	}
}

func (h *Cem) Setup(port, remoteSKI, certFile, keyFile string) error {
	serviceDescription := &service.ServiceDescription{
		Brand:        h.brand,
		Model:        h.model,
		SerialNumber: h.serialNumber,
		Identifier:   h.identifier,
		DeviceType:   model.DeviceTypeTypeEnergyManagementSystem,
	}

	h.myService = service.NewEEBUSService(serviceDescription, h)

	var err error
	var certificate tls.Certificate

	serviceDescription.Port, err = strconv.Atoi(port)
	if err != nil {
		return err
	}

	certificate, err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	serviceDescription.Certificate = certificate

	if err = h.myService.Setup(); err != nil {
		return err
	}

	// Setup the supported UseCases and their features
	measurementsSupport := AddMeasurementSupport(h.myService)
	measurementsSupport.Delegate = h

//	h.hvacSupport = AddHvacSupport(h.myService)
//	h.hvacSupport.Delegate = h

	h.myService.Start()
	defer h.myService.Shutdown()

	remoteService := service.ServiceDetails{
		SKI: remoteSKI,
	}

	h.myService.RegisterRemoteService(remoteService)

	return nil
}

// EEBUSServiceDelegate

// handle a request to trust a remote service
func (h *Cem) RemoteServiceTrustRequested(ski string) {
	// we directly trust it in this example
	h.myService.UpdateRemoteServiceTrust(ski, true)
}

// report the Ship ID of a newly trusted connection
func (h *Cem) RemoteServiceShipIDReported(ski string, shipID string) {
	// we should associated the Ship ID with the SKI and store it
	// so the next connection can start trusted
	fmt.Println("SKI", ski, "has Ship ID:", shipID)
}

// EVSEDelegate

// handle device state updates from the remote EVSE device
func (h *Cem) HandleMeasurement(ski string, details MeasurementDataList) {
	jsVal, err := json.Marshal(details)
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(string(jsVal))
}

func (h *Cem) HandleHvac(ski string, details model.HvacOverrunListDataType) {
	jsVal, err := json.Marshal(details)
	if err != nil {
		fmt.Println(err)
	}
	h.tmpLukas = h.tmpLukas + 1

	fmt.Println(string(jsVal))
	if h.tmpLukas == 2 {
		fmt.Println(">> Now lets bind")
		h.hvacSupport.Bind("d253d08ace1dcfa2bbef88260751e8090cbfc568")
	}
	if h.tmpLukas == 3 {
		fmt.Println(">> Now lets set something")
		h.hvacSupport.SetOverrun("d253d08ace1dcfa2bbef88260751e8090cbfc568", 2)
	}
}
