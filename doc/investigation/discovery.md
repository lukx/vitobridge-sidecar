# Vitoconnect Discovery

This is what the Vitoconnect module exposes as features:

Full message at [./discovery-data.json](./discovery-data.json)

## Function Overview

| Feature | Role | Function | r/w |
|---------|------|----------|-----|
|NodeManagement|special|nodeManagementBindingData|r|
|NodeManagement|special|nodeManagementBindingDeleteCall|-|
|NodeManagement|special|nodeManagementBindingRequestCall|-|
|NodeManagement|special|nodeManagementDetailedDiscoveryData|r|
|NodeManagement|special|nodeManagementSubscriptionData|r|
|NodeManagement|special|nodeManagementSubscriptionDeleteCall|-|
|NodeManagement|special|nodeManagementSubscriptionRequestCall|-|
|NodeManagement|special|nodeManagementUseCaseData|r|
|DeviceClassification|server|deviceClassificationManufacturerData|r|
|ElectricalConnection|server|electricalConnectionDescriptionListData|r|
|ElectricalConnection|server|electricalConnectionParameterDescriptionListData|r|
|HVAC|server|hvacOverrunDescriptionListData|r|
|HVAC|server|hvacOverrunListData|r/w|
|HVAC|server|hvacSystemFunctionDescriptionListData|r|
|Measurement|server|measurementConstraintsListData|r|
|Measurement|server|measurementDescriptionListData|r|
|Measurement|server|measurementListData|r|

## Supported Use Cases

see [./usecase-data.json](./usecase-data.json)

* monitoringAndControlOfSmartGridReadyConditions
* monitoringOfPowerConsumption
