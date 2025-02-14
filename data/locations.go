package data

import (
	"fmt"
	"log"
	"mtgo/tools"
	"path/filepath"

	"github.com/goccy/go-json"
)

// #region Location getters

func GetLocations() *Locations {
	return &db.location.Bases
}

func GetLocationById(id string) *LocationBase {
	location, ok := db.location.Bases.Locations[id]
	if !ok {
		log.Fatal("location doesn't exist")
	}
	return &location
}

// need to populate this bitch
var locationIdByName = map[string]string{}

func GetLocationIdByName(name string) (string, error) {
	id, ok := locationIdByName[name]
	if !ok {
		return "", fmt.Errorf("no location by that name")
	}
	return id, nil
}

// #endregion

// #region Location setters

func setLocations() {
	db.location = &Location{
		Bases: Locations{
			Locations: make(map[string]LocationBase),
			Paths:     make([]Path, 0),
		},
		Loot: make(map[string][][]LootSpawn),
	}

	raw := tools.GetJSONRawMessage(filepath.Join(locationsPath, "locations.json"))
	if err := json.UnmarshalNoEscape(raw, &db.location.Bases); err != nil {
		msg := tools.CheckParsingError(raw, err)
		log.Fatalln(msg)
	}

	directories, _ := tools.GetDirectoriesFrom(locationsPath)
	for dir := range directories {
		directory := filepath.Join(locationsPath, dir)

		baseFilePath := filepath.Join(directory, "base.json")
		id := ""
		if tools.FileExist(baseFilePath) {
			b := tools.GetJSONRawMessage(filepath.Join(directory, "base.json"))
			base := LocationBase{}
			if err := json.UnmarshalNoEscape(b, &base); err != nil {
				msg := tools.CheckParsingError(b, err)
				log.Fatalln(msg)
			}
			id = base.Id
			db.location.Bases.Locations[id] = base

			locationIdByName[base.NameId] = id
		}

		lootFilePath := filepath.Join(directory, "loot.json")
		if tools.FileExist(lootFilePath) {
			l := tools.GetJSONRawMessage(lootFilePath)
			loot := make([][]LootSpawn, 0)
			if err := json.UnmarshalNoEscape(l, &loot); err != nil {
				msg := tools.CheckParsingError(l, err)
				log.Fatalln(msg)
			}
			db.location.Loot[id] = loot
		}
	}
}

//make(map[string][]any)

func GetLocalLootByNameAndIndex(id string, index int8) ([]LootSpawn, error) {
	location, ok := db.location.Loot[id]
	if !ok {
		log.Println("Location", id, "doesn't exist in localLoot map")
		return nil, fmt.Errorf("location %s does not exist", id)
	}

	loot := location[index]
	if loot != nil {
		return loot, nil
	}
	return nil, fmt.Errorf("loot index for %s does not exist", id)
}

// #endregion

type Waves struct {
	Number        int16  `json:"number"`
	TimeMin       int16  `json:"time_min"`
	TimeMax       int16  `json:"time_max"`
	SlotsMin      int16  `json:"slots_min"`
	SlotsMax      int16  `json:"slots_max"`
	SpawnPoints   string `json:"SpawnPoints"`
	BotSide       string `json:"BotSide"`
	BotPreset     string `json:"BotPreset"`
	WildSpawnType string `json:"WildSpawnType"`
	IsPlayers     bool   `json:"isPlayers"`
	OpenZones     string `json:"OpenZones,omitempty"`
}

type NonWaveGroupScenario struct {
	MinToBeGroup int8 `json:"MinToBeGroup"`
	MaxToBeGroup int8 `json:"MaxToBeGroup"`
	Chance       int8 `json:"Chance"`
	Enabled      bool `json:"Enabled"`
}

type Exit struct {
	Name               string `json:"Name"`
	EntryPoints        string `json:"EntryPoints"`
	Chance             int8   `json:"Chance"`
	MinTime            int16  `json:"MinTime"`
	MaxTime            int16  `json:"MaxTime"`
	PlayersCount       int8   `json:"PlayersCount"`
	ExfiltrationTime   int16  `json:"ExfiltrationTime"`
	PassageRequirement string `json:"PassageRequirement,omitempty"`
	ExfiltrationType   string `json:"ExfiltrationType,omitempty"`
	Id                 string `json:"Id"`
	Count              int32  `json:"Count,omitempty"`
	RequirementTip     string `json:"RequirementTip,omitempty"`
}

type BossLocationSpawn struct {
	BossName            string         `json:"BossName"`
	BossChance          int8           `json:"BossChance"`
	BossZone            string         `json:"BossZone"`
	BossPlayer          bool           `json:"BossPlayer"`
	BossDifficult       string         `json:"BossDifficult"`
	BossEscortType      string         `json:"BossEscortType"`
	BossEscortDifficult string         `json:"BossEscortDifficult"`
	BossEscortAmount    string         `json:"BossEscortAmount"`
	Time                int32          `json:"Time"`
	TriggerId           *string        `json:"TriggerId,omitempty"`
	TriggerName         *string        `json:"TriggerName,omitempty"`
	Supports            []*BossSupport `json:"Supports"`
	RandomTimeSpawn     bool           `json:"RandomTimeSpawn"`
}

type BossSupport struct {
	BossEscortType      string   `json:"BossEscortType"`
	BossEscortDifficult []string `json:"BossEscortDifficult"`
	BossEscortAmount    string   `json:"BossEscortAmount"`
}

type SpawnPointParam struct {
	Id                 string         `json:"Id"`
	Position           Vector3        `json:"Position"`
	Rotation           float32        `json:"Rotation"`
	Sides              []string       `json:"Sides"`
	Categories         []string       `json:"Categories"`
	Infiltration       string         `json:"Infiltration"`
	DelayToCanSpawnSec float32        `json:"DelayToCanSpawnSec"`
	ColliderParams     ColliderParams `json:"ColliderParams"`
	CorePointId        float32        `json:"CorePointId,omitempty"`
	BotZoneName        string         `json:"BotZoneName"`
}

type ColliderParams struct {
	Parent string             `json:"_parent"`
	Props  ColliderParamProps `json:"_props"`
}

type ColliderParamProps struct {
	Center Vector3 `json:"Center"`
	Size   Vector3 `json:"Size,omitempty"`
	Radius float32 `json:"Radius,omitempty"`
}

type AirdropParameter struct {
	PlaneAirdropStartMin           int16   `json:"PlaneAirdropStartMin"`
	PlaneAirdropStartMax           int16   `json:"PlaneAirdropStartMax"`
	PlaneAirdropEnd                int16   `json:"PlaneAirdropEnd"`
	PlaneAirdropChance             float32 `json:"PlaneAirdropChance"`
	PlaneAirdropMax                int16   `json:"PlaneAirdropMax"`
	PlaneAirdropCooldownMin        int16   `json:"PlaneAirdropCooldownMin"`
	PlaneAirdropCooldownMax        int16   `json:"PlaneAirdropCooldownMax"`
	AirdropPointDeactivateDistance int16   `json:"AirdropPointDeactivateDistance"`
	MinPlayersCountToSpawnAirdrop  int8    `json:"MinPlayersCountToSpawnAirdrop"`
	UnsuccessfulTryPenalty         int16   `json:"UnsuccessfulTryPenalty"`
}

type LootSpawn struct {
	Id              string                       `json:"Id"`
	IsContainer     bool                         `json:"IsContainer"`
	UseGravity      bool                         `json:"useGravity"`     //used for loose loot
	RandomRotation  bool                         `json:"randomRotation"` //used for loose loot
	Position        Vector3                      `json:"Position"`
	Rotation        Vector3                      `json:"Rotation"`
	IsGroupPosition bool                         `json:"IsGroupPosition"` //dynamic container
	GroupPositions  []*WeightedLootSpawnPosition `json:"GroupPositions"`  //dynamic container spawn positions
	IsAlwaysSpawn   bool                         `json:"IsAlwaysSpawn"`
	Root            string                       `json:"Root"`  //main container id
	Items           []InventoryItem              `json:"Items"` //items in the spawn
}

type WeightedLootSpawnPosition struct {
	Name     string  `json:"Name"` //string.Format("groupPoint[{0}]", this._groupPositions.Count)
	Weight   int8    `json:"Weight"`
	Position Vector3 `json:"Position"`
	Rotation Vector3 `json:"Rotation"`
}

type Banner struct {
	Id  string `json:"id"`
	Pic Prefab `json:"pic"`
}

type BotLocationModifier struct {
	AccuracySpeed          float32 `json:"AccuracySpeed"`
	DistToActivate         float32 `json:"DistToActivate"`
	DistToPersueAxemanCoef float32 `json:"DistToPersueAxemanCoef"`
	DistToSleep            float32 `json:"DistToSleep"`
	GainSight              float32 `json:"GainSight"`
	KhorovodChance         float32 `json:"KhorovodChance"`
	MagnetPower            float32 `json:"MagnetPower"`
	MarksmanAccuratyCoef   float32 `json:"MarksmanAccuratyCoef"`
	Scattering             float32 `json:"Scattering"`
	VisibleDistance        float32 `json:"VisibleDistance"`
}

type MinPlayersByWaitTime struct {
	MinPlayers int8  `json:"minPlayers"`
	Time       int16 `json:"time"`
}

type MinMaxBots struct {
	WildSpawnType string `json:"WildSpawnType"`
	Max           int8   `json:"max"`
	Min           int8   `json:"min"`
}

type MaxOfItemAllowedOnLocation struct {
	TemplateId string `json:"TemplateId"`
	Value      int32  `json:"Value"`
}

type Limit struct {
	Items []string `json:"items"`
	Min   int8     `json:"min"`
	Max   int8     `json:"max"`
}

// #region Location structs
type Location struct {
	Bases Locations
	Loot  map[string][][]LootSpawn
}

type Locations struct {
	Locations map[string]LocationBase `json:"locations"`
	Paths     []Path                  `json:"paths"`
}

type Path struct {
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
}

type LocationBase struct {
	AccessKeys                     []*string                    `json:"AccessKeys"`
	AirdropParameters              []*AirdropParameter          `json:"AirdropParameters,omitempty"`
	Area                           float32                      `json:"Area"`
	AveragePlayTime                int32                        `json:"AveragePlayTime"`
	AveragePlayerLevel             int8                         `json:"AveragePlayerLevel"`
	Banners                        []Banner                     `json:"Banners"`
	BossLocationSpawn              []BossLocationSpawn          `json:"BossLocationSpawn"`
	BotAssault                     int8                         `json:"BotAssault"`
	BotEasy                        int8                         `json:"BotEasy"`
	BotHard                        int8                         `json:"BotHard"`
	BotImpossible                  int8                         `json:"BotImpossible"`
	BotLocationModifier            BotLocationModifier          `json:"BotLocationModifier"`
	BotMarksman                    int8                         `json:"BotMarksman"`
	BotMax                         int8                         `json:"BotMax"`
	BotMaxPlayer                   int8                         `json:"BotMaxPlayer"`
	BotMaxTimePlayer               int16                        `json:"BotMaxTimePlayer"`
	BotNormal                      int8                         `json:"BotNormal"`
	BotSpawnCountStep              int8                         `json:"BotSpawnCountStep"`
	BotSpawnPeriodCheck            int8                         `json:"BotSpawnPeriodCheck"`
	BotSpawnTimeOffMax             int8                         `json:"BotSpawnTimeOffMax"`
	BotSpawnTimeOffMin             int8                         `json:"BotSpawnTimeOffMin"`
	BotSpawnTimeOnMax              int16                        `json:"BotSpawnTimeOnMax"`
	BotSpawnTimeOnMin              int16                        `json:"BotSpawnTimeOnMin"`
	BotStart                       int16                        `json:"BotStart"`
	BotStartPlayer                 int16                        `json:"BotStartPlayer"`
	BotStop                        int16                        `json:"BotStop"`
	Description                    string                       `json:"Description"`
	DisabledForScav                bool                         `json:"DisabledForScav"`
	DisabledScavExits              string                       `json:"DisabledScavExits"`
	EnableCoop                     bool                         `json:"EnableCoop"`
	Enabled                        bool                         `json:"Enabled"`
	EscapeTimeLimit                int32                        `json:"EscapeTimeLimit"`
	EscapeTimeLimitCoop            int32                        `json:"EscapeTimeLimitCoop"`
	GenerateLocalLootCache         bool                         `json:"GenerateLocalLootCache"`
	GlobalContainerChanceModifier  float32                      `json:"GlobalContainerChanceModifier"`
	GlobalLootChanceModifier       float32                      `json:"GlobalLootChanceModifier"`
	IconX                          int16                        `json:"IconX"`
	IconY                          int16                        `json:"IconY"`
	NameId                         string                       `json:"Id"`
	Insurance                      bool                         `json:"Insurance,omitempty"`
	IsSecret                       bool                         `json:"IsSecret"`
	Locked                         bool                         `json:"Locked"`
	Loot                           []LootSpawn                  `json:"Loot"`
	MatchMakerMinPlayersByWaitTime []MinPlayersByWaitTime       `json:"MatchMakerMinPlayersByWaitTime"`
	MaxBotPerZone                  int8                         `json:"MaxBotPerZone"`
	MaxCoopGroup                   int8                         `json:"MaxCoopGroup,omitempty"`
	MaxDistToFreePoint             int16                        `json:"MaxDistToFreePoint"`
	MaxPlayers                     int8                         `json:"MaxPlayers"`
	MinDistToExitPoint             int16                        `json:"MinDistToExitPoint"`
	MinDistToFreePoint             int16                        `json:"MinDistToFreePoint"`
	MinMaxBots                     []MinMaxBots                 `json:"MinMaxBots"`
	MinPlayerLvlAccessKeys         int8                         `json:"MinPlayerLvlAccessKeys"`
	MinPlayers                     int8                         `json:"MinPlayers"`
	Name                           string                       `json:"Name"`
	NewSpawn                       bool                         `json:"NewSpawn"`
	NonWaveGroupScenario           NonWaveGroupScenario         `json:"NonWaveGroupScenario"`
	OcculsionCullingEnabled        bool                         `json:"OcculsionCullingEnabled"`
	OfflineNewSpawn                bool                         `json:"OfflineNewSpawn"`
	OfflineOldSpawn                bool                         `json:"OfflineOldSpawn"`
	OldSpawn                       bool                         `json:"OldSpawn"`
	OpenZones                      string                       `json:"OpenZones"`
	PlayersRequestCount            int8                         `json:"PlayersRequestCount"`
	PmcMaxPlayersInGroup           int8                         `json:"PmcMaxPlayersInGroup"`
	Preview                        Prefab                       `json:"Preview"`
	RequiredPlayerLevelMax         int8                         `json:"RequiredPlayerLevelMax"`
	RequiredPlayerLevelMin         int8                         `json:"RequiredPlayerLevelMin"`
	Rules                          string                       `json:"Rules"`
	SafeLocation                   bool                         `json:"SafeLocation,omitempty"`
	ScavMaxPlayersInGroup          int8                         `json:"ScavMaxPlayersInGroup"`
	Scene                          Prefab                       `json:"Scene"`
	SpawnPointParams               []SpawnPointParam            `json:"SpawnPointParams"` //needs to be checked
	UnixDateTime                   int32                        `json:"UnixDateTime"`
	Id                             string                       `json:"_Id"`
	Doors                          []any                        `json:"doors"`
	ExitAccessTime                 int16                        `json:"exit_access_time"`
	ExitCount                      int8                         `json:"exit_count,omitempty"`
	ExitTime                       int8                         `json:"exit_time"`
	Exits                          []Exit                       `json:"exits"`
	FilterEx                       []string                     `json:"filter_ex"`
	Limits                         []Limit                      `json:"limits"`
	MatchingMinSeconds             int8                         `json:"matching_min_seconds"`
	MaxItemCountInLocation         []MaxOfItemAllowedOnLocation `json:"maxItemCountInLocation"`
	SavSummonSeconds               int8                         `json:"sav_summon_seconds"`
	TmpLocationFieldRemoveMe       int8                         `json:"tmp_location_field_remove_me"`
	UsersGatherSeconds             int16                        `json:"users_gather_seconds"`
	UsersSpawnSecondsN             int16                        `json:"users_spawn_seconds_n"`
	UsersSpawnSecondsN2            int16                        `json:"users_spawn_seconds_n2"`
	UsersSummonSeconds             int16                        `json:"users_summon_seconds"`
	Waves                          []Waves                      `json:"waves"`
}

// #endregion
