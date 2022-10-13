package cem

import (
	"fmt"
	"time"

	"github.com/DerAndereAndi/eebus-go/service"
	"github.com/DerAndereAndi/eebus-go/spine"
	"github.com/DerAndereAndi/eebus-go/spine/model"
)

// Delegate Interface for the EVSE
type HvacDelegate interface {

	// handle device manufacturer data updates from the remote EVSE device
	HandleHvac(ski string, details model.HvacOverrunListDataType)
}

type Hvac struct {
	*spine.UseCaseImpl

	service *service.EEBUSService

	Delegate HvacDelegate
	quitLoop chan struct{}
}

// Add EVSE support
func AddHvacSupport(service *service.EEBUSService) *Hvac {
	entity := service.LocalEntity()

	// add the use case
	useCase := &Hvac{
		UseCaseImpl: spine.NewUseCase(
			entity,
			model.UseCaseNameTypeMonitoringAndControlOfSmartGridReadyConditions,
			"1.0.0",
			[]model.UseCaseScenarioSupportType{1, 2}),
		service: service,
	}
	spine.Events.Subscribe(useCase)

	{
		f := service.LocalEntity().GetOrAddFeature(model.FeatureTypeTypeHvac, model.RoleTypeClient, "Hvac Client")
		entity.AddFeature(f)
	}

	return useCase
}

// Internal EventHandler Interface for the CEM
func (e *Hvac) HandleEvent(payload spine.EventPayload) {
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
		case model.SubscriptionManagementRequestCallType:
			data := payload.Data.(model.SubscriptionManagementRequestCallType)
			if *data.ServerFeatureType == model.FeatureTypeTypeHvac {
				remoteDevice := e.service.RemoteDeviceForSki(payload.Ski)
				if remoteDevice == nil {
					fmt.Println("No remote device found for SKI:", payload.Ski)
					return
				}
				switch payload.ChangeType {
				case spine.ElementChangeAdd:
					// start sending heartbeats
					senderAddr := e.Entity.Device().FeatureByTypeAndRole(model.FeatureTypeTypeHvac, model.RoleTypeServer).Address()
					rEntity := remoteDevice.Entity([]model.AddressEntityType{1})
					destinationAddr := remoteDevice.FeatureByEntityTypeAndRole(rEntity, model.FeatureTypeTypeHvac, model.RoleTypeClient).Address()
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
			case *model.HvacOverrunDataType:
				hvacOverrunData := payload.Data.(model.HvacOverrunDataType)
				fmt.Println(hvacOverrunData.OverrunId)
			}
		}
	}
}

func (e *Hvac) startMeasuringDevice(remoteDevice *spine.DeviceRemoteImpl) {
	ticker := time.NewTicker(5 * time.Second)
	e.quitLoop = make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				// todo: could work for more devices
				e.requestHvac(remoteDevice)
			case <-e.quitLoop:
				ticker.Stop()
				return
			}
		}
	}()
}

func (e *Hvac) stopMeasuringDevice(remoteDevice *spine.DeviceRemoteImpl) {
	close(e.quitLoop)
}

// request DeviceDiagnosisStateData from a remote device
func (e *Hvac) requestHvac(remoteDevice *spine.DeviceRemoteImpl) {
	rEntity := remoteDevice.Entity([]model.AddressEntityType{1})

	hvacValues := requestHvacForEntity(e.service, rEntity)

	if hvacValues != nil && e.Delegate != nil {
		e.Delegate.HandleHvac(remoteDevice.Ski(), *hvacValues)
	}
}

func (e *Hvac) Bind(remoteSki string) {

	remoteDevice := e.service.RemoteDeviceForSki(remoteSki)
	rEntity := remoteDevice.Entity([]model.AddressEntityType{1})

	featureLocal, featureRemote, err := e.service.GetLocalClientAndRemoteServerFeatures(model.FeatureTypeTypeHvac, rEntity)
	if err != nil {
		fmt.Println(err)
		return
	}

	_, nmgtFeature, err := e.service.GetLocalClientAndRemoteServerFeatures(model.FeatureTypeTypeNodeManagement, rEntity)
	if err != nil {
		fmt.Println(err)
		return
	}

	featureType := model.FeatureTypeTypeHvac
	bindingRequest := model.BindingManagementRequestCallType{
		ClientAddress:     featureLocal.Address(),
		ServerAddress:     featureRemote.Address(),
		ServerFeatureType: &featureType,
	}

	featureLocal.WriteData(model.FunctionTypeNodeManagementBindingData, bindingRequest, nmgtFeature)
}

func (e *Hvac) SetOverrun(remoteSki string, enabledId int) {

	remoteDevice := e.service.RemoteDeviceForSki(remoteSki)
	rEntity := remoteDevice.Entity([]model.AddressEntityType{1})

	featureLocal, featureRemote, err := e.service.GetLocalClientAndRemoteServerFeatures(model.FeatureTypeTypeHvac, rEntity)
	if err != nil {
		fmt.Println(err)
		return
	}
	featureLocal.WriteData(model.FunctionTypeHvacOverrunListData, generateHvacOverrunListDataType(enabledId), featureRemote)
}

func generateHvacOverrunListDataType(enabledId int) *model.HvacOverrunListDataType {
	newOverrun := model.HvacOverrunListDataType{
		HvacOverrunData: []model.HvacOverrunDataType{},
	}
	for i := 0; i < 3; i++ {
		var status model.HvacOverrunStatusType
		if enabledId == i {
			status = model.HvacOverrunStatusTypeActive
		} else {
			status = model.HvacOverrunStatusTypeInactive
		}

		idType := model.HvacOverrunIdType(i)

		newOverrun.HvacOverrunData = append(newOverrun.HvacOverrunData, model.HvacOverrunDataType{
			OverrunId:     &idType,
			OverrunStatus: &status,
		})
	}

	return &newOverrun
}

func requestHvacForEntity(service *service.EEBUSService, entity *spine.EntityRemoteImpl) *model.HvacOverrunListDataType {
	featureLocal, featureRemote, err := service.GetLocalClientAndRemoteServerFeatures(model.FeatureTypeTypeHvac, entity)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	msgCounter, errrr := featureLocal.RequestData(model.FunctionTypeHvacOverrunListData, featureRemote)
	if errrr != nil {
		fmt.Println(err)
		return nil
	}

	result, errrr := featureLocal.FetchRequestData(*msgCounter, featureRemote)
	if errrr != nil {
		fmt.Println(err)
		return nil
	}

	response := result.(*model.HvacOverrunListDataType)

	return response
}

func requestHvacDescriptionForEntity(service *service.EEBUSService, entity *spine.EntityRemoteImpl) *model.HvacOverrunDescriptionListDataType {
	featureLocal, featureRemote, err := service.GetLocalClientAndRemoteServerFeatures(model.FeatureTypeTypeHvac, entity)
	if err != nil {
		fmt.Println(err)
		return nil
	}

	msgCounter, errrr := featureLocal.RequestData(model.FunctionTypeHvacOverrunDescriptionListData, featureRemote)
	if errrr != nil {
		fmt.Println(errrr)
		return nil
	}

	result, errrr := featureLocal.FetchRequestData(*msgCounter, featureRemote)
	if errrr != nil {
		fmt.Println(errrr)
		return nil
	}

	response := result.(*model.HvacOverrunDescriptionListDataType)

	return response
}
