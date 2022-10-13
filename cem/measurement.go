package cem

import (
	"fmt"
	"time"

	"github.com/DerAndereAndi/eebus-go/service"
	"github.com/DerAndereAndi/eebus-go/spine"
	"github.com/DerAndereAndi/eebus-go/spine/model"
)

type MeasurementData struct {
	Value       *model.MeasurementDataType            `json:"value,omitempty"`
	Description *model.MeasurementDescriptionDataType `json:"description,omitempty"`
}

type MeasurementDataList struct {
	MeasurementData []MeasurementData `json:"value,measurements"`
}

// Delegate Interface for the EVSE
type MeasurementDelegate interface {

	// handle device manufacturer data updates from the remote EVSE device
	HandleMeasurement(ski string, details MeasurementDataList)
}

type Measurement struct {
	*spine.UseCaseImpl

	service *service.EEBUSService

	Delegate MeasurementDelegate

	// map of device SKIs to descriptors
	knownDescriptions map[string]*model.MeasurementDescriptionListDataType
	quitLoop          chan struct{}
}

// Add EVSE support
func AddMeasurementSupport(service *service.EEBUSService) *Measurement {
	entity := service.LocalEntity()

	// add the use case
	useCase := &Measurement{
		UseCaseImpl: spine.NewUseCase(
			entity,
			model.UseCaseNameTypeMonitoringOfPowerConsumption,
			"1.0.0",
			[]model.UseCaseScenarioSupportType{1, 2, 3, 4}),
		service: service,
		knownDescriptions: map[string]*model.MeasurementDescriptionListDataType{},
	}
	spine.Events.Subscribe(useCase)

	{
		f := service.LocalEntity().GetOrAddFeature(model.FeatureTypeTypeMeasurement, model.RoleTypeClient, "Measurement Client")
		entity.AddFeature(f)
	}

	return useCase
}

// Internal EventHandler Interface for the CEM
func (e *Measurement) HandleEvent(payload spine.EventPayload) {
	fmt.Println(payload.EventType)
	switch payload.EventType {
	case spine.EventTypeDeviceChange:
		switch payload.ChangeType {
		case spine.ElementChangeAdd:
			e.startMeasuringDevice(payload.Device)
		case spine.ElementChangeRemove:
			e.stopMeasuringDevice(payload.Device)
		}
	case spine.EventTypeSubscriptionChange:
		switch payload.Data.(type) {
		// todo: this section is irrelevant but I am keeping it for easier copy & paste.
		// this would only happen if we were offering measurement in Server mode.
		case model.SubscriptionManagementRequestCallType:
			data := payload.Data.(model.SubscriptionManagementRequestCallType)
			if *data.ServerFeatureType == model.FeatureTypeTypeMeasurement {
				remoteDevice := e.service.RemoteDeviceForSki(payload.Ski)
				if remoteDevice == nil {
					fmt.Println("No remote device found for SKI:", payload.Ski)
					return
				}
				switch payload.ChangeType {
				case spine.ElementChangeAdd:
					// start sending heartbeats
					senderAddr := e.Entity.Device().FeatureByTypeAndRole(model.FeatureTypeTypeMeasurement, model.RoleTypeServer).Address()
					rEntity := remoteDevice.Entity([]model.AddressEntityType{1})
					destinationAddr := remoteDevice.FeatureByEntityTypeAndRole(rEntity, model.FeatureTypeTypeMeasurement, model.RoleTypeClient).Address()
					if senderAddr == nil || destinationAddr == nil {
						fmt.Println("No sender or destination address found for SKI:", payload.Ski)
						return
					}
					remoteDevice := e.service.RemoteDeviceForSki(payload.Ski)
					remoteDevice.StartHeartbeatSend(senderAddr, destinationAddr)
				}
			}
		}

	case spine.EventTypeDataChange:
		if payload.ChangeType == spine.ElementChangeUpdate {
			switch payload.Data.(type) {
			case *model.MeasurementListDataType:
				measurementData := payload.Data.(*model.MeasurementListDataType)
				fmt.Printf(
					"Received Measurement %s", measurementData.MeasurementData[0].Value,
				)
				//mdl := interlinkMeasurementsWithDescriptions(measurementData, )
				//e.Delegate.HandleMeasurement(payload.Ski, measurementData)
			}
		}
	}
}

func (e *Measurement) startMeasuringDevice(remoteDevice *spine.DeviceRemoteImpl) {
	rEntity := remoteDevice.Entity([]model.AddressEntityType{1})
	subscribeToMeasurements(e.service, rEntity)

	return
	// Note: currently attempting to use the subscription feature , therefore an early return here

	ticker := time.NewTicker(5 * time.Second)
	e.quitLoop = make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				// todo: could work for more devices
				e.requestMeasurements(remoteDevice)
			case <-e.quitLoop:
				ticker.Stop()
				return
			}
		}
	}()
}

func (e *Measurement) stopMeasuringDevice(remoteDevice *spine.DeviceRemoteImpl) {
	close(e.quitLoop)
}

// request DeviceDiagnosisStateData from a remote device
func (e *Measurement) requestMeasurements(remoteDevice *spine.DeviceRemoteImpl) {
	rEntity := remoteDevice.Entity([]model.AddressEntityType{1})
	var descriptions *model.MeasurementDescriptionListDataType
	var ok bool

	if descriptions, ok = e.knownDescriptions[remoteDevice.Ski()]; !ok {
		descriptions = requestMeasurementDescriptionsForEntity(e.service, rEntity)
		if descriptions == nil {

			return

		}
		fmt.Printf("Storing descriptions for this SKI")
		e.knownDescriptions[remoteDevice.Ski()] = descriptions
	}

	measurements := requestMeasurementsForEntity(e.service, rEntity)

	if measurements == nil {
		return
	}

	annotatedMeasurements := interlinkMeasurementsWithDescriptions(measurements, descriptions)

	if e.Delegate != nil {
		e.Delegate.HandleMeasurement(remoteDevice.Ski(), *annotatedMeasurements)
	}

}

func interlinkMeasurementsWithDescriptions(measurements *model.MeasurementListDataType, descriptions *model.MeasurementDescriptionListDataType) *MeasurementDataList {
	list := MeasurementDataList{
		MeasurementData: []MeasurementData{},
	}

	for _, e := range measurements.MeasurementData {
		measurementData := e
		thisData := MeasurementData{
			Description: pickMeasurementDescriptor(measurementData.MeasurementId, descriptions),
			Value:       &measurementData,
		}
		list.MeasurementData = append(list.MeasurementData, thisData)
	}

	return &list
}

func pickMeasurementDescriptor(measurementId *model.MeasurementIdType, descriptions *model.MeasurementDescriptionListDataType) *model.MeasurementDescriptionDataType {
	for _, description := range descriptions.MeasurementDescriptionData {
		if *measurementId == *description.MeasurementId {
			return &description
		}
	}
	return nil
}

func subscribeToMeasurements(service *service.EEBUSService, entity *spine.EntityRemoteImpl)  {
	featureLocal, featureRemote, err := service.GetLocalClientAndRemoteServerFeatures(model.FeatureTypeTypeMeasurement, entity)
	if err != nil {
		fmt.Println(err)
		return
	}
	fErr := featureLocal.SubscribeAndWait(featureRemote.Device(), featureRemote.Address())
	if fErr != nil {
		fmt.Println(fErr.String())
	}

}

func requestMeasurementsForEntity(service *service.EEBUSService, entity *spine.EntityRemoteImpl) *model.MeasurementListDataType {
	featureLocal, featureRemote, err := service.GetLocalClientAndRemoteServerFeatures(model.FeatureTypeTypeMeasurement, entity)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	msgCounter, errrr := featureLocal.RequestData(model.FunctionTypeMeasurementListData, featureRemote)
	if errrr != nil {
		fmt.Println(errrr)
		return nil
	}

	result, errrr := featureLocal.FetchRequestData(*msgCounter, featureRemote)
	if errrr != nil {
		fmt.Println(errrr)
		return nil
	}

	response := result.(*model.MeasurementListDataType)

	return response
}

func requestMeasurementDescriptionsForEntity(service *service.EEBUSService, entity *spine.EntityRemoteImpl) *model.MeasurementDescriptionListDataType {
	featureLocal, featureRemote, err := service.GetLocalClientAndRemoteServerFeatures(model.FeatureTypeTypeMeasurement, entity)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	msgCounter, errrr := featureLocal.RequestData(model.FunctionTypeMeasurementDescriptionListData, featureRemote)
	if errrr != nil {
		fmt.Println(errrr)
		return nil
	}

	result, errrr := featureLocal.FetchRequestData(*msgCounter, featureRemote)
	if errrr != nil {
		fmt.Println(errrr)
		return nil
	}

	response := result.(*model.MeasurementDescriptionListDataType)
	return response
}
