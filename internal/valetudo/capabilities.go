package valetudo

type Capability string

const (
	CapBasicControl                       Capability = "BasicControlCapability"
	CapFanSpeedControl                    Capability = "FanSpeedControlCapability"
	CapWaterUsageControl                  Capability = "WaterUsageControlCapability"
	CapWifiConfiguration                 Capability = "WifiConfigurationCapability"
	CapWifiScan                           Capability = "WifiScanCapability"
	CapZoneCleaning                       Capability = "ZoneCleaningCapability"
	CapMapSegmentation                    Capability = "MapSegmentationCapability"
	CapDoNotDisturb                       Capability = "DoNotDisturbCapability"
	CapConsumableMonitoring               Capability = "ConsumableMonitoringCapability"
	CapLocate                             Capability = "LocateCapability"
	CapGoToLocation                       Capability = "GoToLocationCapability"
	CapCarpetModeControl                  Capability = "CarpetModeControlCapability"
	CapMapReset                           Capability = "MapResetCapability"
	CapMapSegmentEdit                     Capability = "MapSegmentEditCapability"
	CapMapSegmentRename                   Capability = "MapSegmentRenameCapability"
	CapSpeakerTest                        Capability = "SpeakerTestCapability"
	CapSpeakerVolumeControl               Capability = "SpeakerVolumeControlCapability"
	CapVoicePackManagement                Capability = "VoicePackManagementCapability"
	CapCombinedVirtualRestrictions        Capability = "CombinedVirtualRestrictionsCapability"
	CapPendingMapChangeHandling           Capability = "PendingMapChangeHandlingCapability"
	CapMappingPass                        Capability = "MappingPassCapability"
	CapKeyLock                            Capability = "KeyLockCapability"
	CapAutoEmptyDockManualTrigger         Capability = "AutoEmptyDockManualTriggerCapability"
	CapMopDockCleanManualTrigger          Capability = "MopDockCleanManualTriggerCapability"
	CapMopDockDryManualTrigger            Capability = "MopDockDryManualTriggerCapability"
	CapOperationModeControl               Capability = "OperationModeControlCapability"
	CapObstacleAvoidanceControl           Capability = "ObstacleAvoidanceControlCapability"
	CapPetObstacleAvoidanceControl        Capability = "PetObstacleAvoidanceControlCapability"
	CapCarpetSensorModeControl            Capability = "CarpetSensorModeControlCapability"
	CapCollisionAvoidantNavigationControl Capability = "CollisionAvoidantNavigationControlCapability"
	CapTotalStatistics                    Capability = "TotalStatisticsCapability"
	CapCurrentStatistics                  Capability = "CurrentStatisticsCapability"
	CapMopDockMopWashTemperatureControl   Capability = "MopDockMopWashTemperatureControlCapability"
	CapMopDockMopDryingTimeControl        Capability = "MopDockMopDryingTimeControlCapability"
	CapMopExtensionControl                Capability = "MopExtensionControlCapability"
)

type CapabilitySet struct {
	caps map[Capability]bool
}

func NewCapabilitySet(list []string) *CapabilitySet {
	cs := &CapabilitySet{
		caps: make(map[Capability]bool, len(list)),
	}
	for _, c := range list {
		cs.caps[Capability(c)] = true
	}
	return cs
}

func (cs *CapabilitySet) Has(cap Capability) bool {
	if cs == nil || cs.caps == nil {
		return false
	}
	return cs.caps[cap]
}

func (cs *CapabilitySet) HasAny(caps ...Capability) bool {
	if cs == nil || cs.caps == nil {
		return false
	}
	for _, c := range caps {
		if cs.caps[c] {
			return true
		}
	}
	return false
}

// HasStation returns true if the robot supports any dock/station capability
func (cs *CapabilitySet) HasStation() bool {
	return cs.HasAny(
		CapAutoEmptyDockManualTrigger,
		CapMopDockCleanManualTrigger,
		CapMopDockDryManualTrigger,
		CapMopDockMopWashTemperatureControl,
		CapMopDockMopDryingTimeControl,
	)
}

// List returns all capabilities as string slice
func (cs *CapabilitySet) List() []string {
	if cs == nil || cs.caps == nil {
		return nil
	}
	res := make([]string, 0, len(cs.caps))
	for c := range cs.caps {
		res = append(res, string(c))
	}
	return res
}
