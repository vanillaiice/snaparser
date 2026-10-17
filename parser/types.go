package parser

type Account struct {
	BasicInformation  BasicInformation  `json:"Basic Information"`
	DeviceInformation DeviceInformation `json:"Device Information"`
	DeviceHistory     []DeviceHistory   `json:"Device History"`
	LoginHistory      []LoginHistory    `json:"Login History"`
}

type BasicInformation struct {
	Username       string
	Name           string
	CreationDate   string `json:"Creation Date"`
	RegistrationIP string `json:"Registration IP"`
	Country        string
	LastActive     string `json:"Last Active"`
}

type DeviceInformation struct {
	Make           string
	ModelID        string `json:"Model ID"`
	ModelName      string `json:"Model Name"`
	Language       string
	OSType         string `json:"OS Type"`
	OSVersion      string `json:"OS Version"`
	ConnectionType string `json:"Connection Type"`
}

type DeviceHistory struct {
	Make       string
	Model      string
	StartTime  string `json:"Start Time"`
	DeviceType string `json:"Device Type"`
}

// TODO
type PrivacyPolicyAndTermsOfServiceAcceptanceHistory struct{}

// TODO
type CustomCreativeToolsTerms struct{}

type LoginHistory struct {
	IP      string
	Country string
	Created string
	Status  string
	Device  string
}

// TODO
type FamilyCenter struct{}

type AccountHistory struct {
	DisplayNameChange       []DisplayNameChange       `json:"Display Name Change"`
	EmailChange             []EmailChange             `json:"Email Change"`
	MobileNumberChange      []MobileNumberChange      `json:"Mobile Number Change"`
	PasswordChange          []PasswordChange          `json:"Password Change"`
	SnapchatLinkedToBitmoji []SnapchatLinkedToBitmoji `json:"Snapchat Linked To Bitmoji"`
	DownloadMyDataReports   []DownloadMyDataReports   `json:"Download My Data Reports"`
}

type DisplayNameChange struct {
	Date        string
	DisplayName string `json:"Display Name"`
}

type EmailChange struct {
	Date         string
	EmailAddress string `json:"Email Address"`
}

type MobileNumberChange struct {
	Date         string
	MobileNumber string `json:"Mobile Number"`
}

type PasswordChange struct {
	Date string
}

type SnapchatLinkedToBitmoji struct {
	Date string
}

// TODO
type Spectacles struct{}

// TODO
type TwoFactorAuthentication struct{}

// TODO
type AccountDeactivatedReactivated struct{}

type DownloadMyDataReports struct {
	Date         string
	Status       string
	EmailAddress string `json:"Email Address"`
}

type ChatHistory map[string][]Chat

type Chat struct {
	From               string
	MediaType          string `json:"Media Type"`
	Created            string
	Content            *string `json:"Content"`
	ConversationTitle  *string `json:"Conversation Title"`
	IsSender           bool    `json:"isSender"`
	CreatedMicrosecond int64   `json:"Created(microseconds)"`
	IsSaved            bool    `json:"IsSaved"`
	MediaIDS           string  `json:"Media IDs"`
}

type Friends struct {
	Friends                 []Friend
	FriendsRequestsSent     []Friend `json:"Friend Requests Sent"`
	BlockedUsers            []Friend `json:"Blocked Users"`
	DeletedFriends          []Friend `json:"Deleted Friends"`
	HiddenFriendSuggestions []Friend `json:"Hidden Friend Suggestions"`
	IgnoredSnapchatters     []Friend `json:"Ignored Snapchatters"`
	PendingRequests         []Friend `json:"Pending Requests"`
	Shortcuts               []Shortcuts
}

type Friend struct {
	Username              string
	DisplayName           string `json:"Display Name"`
	CreationTimestamp     string `json:"Creation Timestamp"`
	LastModifiedTimestamp string `json:"Last Modified Timestamp"`
	Source                string
}

// TODO
type Shortcuts struct{}

type SnapHistory map[string][]Snap

type Snap struct {
	From                string
	MediaType           string `json:"Media Type"`
	Created             string
	ConversationTitle   *string `json:"Conversation Title"`
	IsSender            bool    `json:"IsSender"`
	CreatedMicroseconds int64   `json:"Created(microseconds)"`
}

type SnapProfile struct {
	Profile      Profile
	Spotlights   []Spotlight
	SavedStories []SavedStories `json:"Saved Stories"`
}

type Profile struct {
	Created        string
	ProfileTitle   string `json:"Profile Title"`
	Location       string
	ProfileWebsite string `json:"Profile Website"`
	ProfilePhoto   string `json:"Profile Photo"`
	HeroImage      string `json:"Hero Image"`
}

// TODO
type Spotlight struct{}

// TODO
type SavedStories struct{}

type TalkHistory struct {
	OutgoingCalls  []Call          `json:"Outgoing Calls"`
	IncomingCalls  []Call          `json:"Incoming Calls"`
	CompletedCalls []CompletedCall `json:"Completed Calls"`
	ChatSessions   []ChatSession   `json:"Chat Sessions"`
	GameSessions   []GameSession   `json:"Game Sessions"`
}

type Call struct {
	DateTime     string `json:"Date & Time"`
	Type         string
	PeopleInChat int32 `json:"People in Chat"`
	Result       string
	City         string
	Country      string
	Length       int32 `json:"Length (sec)"`
	Network      string
}

// TODO
type CompletedCall struct{}

// TODO
type ChatSession struct{}

// TODO
type GameSession struct{}

type UserProfile struct {
	AppProfile                AppProfile `json:"App Profile"`
	Demographics              Demographics
	Engagement                []Engagement
	DiscoverChannelsViewed    []DiscoverChannelsViewed `json:"Discover Channels Viewed"`
	BreakdownOfTimeSpentOnApp []string                 `json:"Breakdown of Time Spent on App"`
	AdsYouInteractedWith      []AdsYouInteractedWith   `json:"Ads You Interacted With"`
	InterestCategories        []InterestCategories     `json:"Interest Categories"`
	ContentCategories         []string                 `json:"Content Categories"`
	GeographicInformation     []GeographicInformation  `json:"Geographic Information"`
	Interactions              Interactions             `json:"Interactions"`
	OffPlatformSharing        []OffPlatformSharing     `json:"Off-Platform Sharing"`
	MobileAdID                string                   `json:"Mobile Ad Id"`
}

type AppProfile struct {
	Country                string
	CreationTime           string `json:"Creation Time"`
	AccountCreationCountry string `json:"Account Creation Country"`
	PlatformVersion        string `json:"Platform Version"`
	InAppLanguage          string `json:"In-app Language"`
}

type Demographics struct {
	CohortAge            string `json:"Cohort Age"`
	DerivedAdDemographic string `json:"Derived Ad Demographic"`
}

// TODO (no subs because im not a chud)
type Subscriptions struct{}

type Engagement struct {
	Event       string
	Occurrences int32
}

type DiscoverChannelsViewed struct {
	ChannelName string `json:"Channel Name"`
}

type AdsYouInteractedWith struct {
	AdvertiserName string `json:"Advertiser Name"`
	Date           string
}

// TODO
type InterestCategories struct{}

// TODO
type GeographicInformation struct{}

type Interactions struct {
	WebInteractions []string `json:"Web Interactions"`
	AppInteractions []string `json:"App Interactions"`
}

// TODO
type OffPlatformSharing struct{}
