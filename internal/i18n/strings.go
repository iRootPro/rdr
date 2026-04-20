// Package i18n holds the UI string tables for rdr. Translations are
// typed structs (not maps) so every reference is compile-time checked and
// a missing field is caught by TestAllFieldsFilled in strings_test.go.
package i18n

type Lang string

const (
	EN Lang = "en"
	RU Lang = "ru"
)

// Strings is the top-level translation table, grouped by UI zone.
type Strings struct {
	Settings SettingsStrings
	Help     HelpStrings
	Keys     KeysStrings
	Toasts   ToastsStrings
	Status   StatusStrings
	Reader   ReaderStrings
	Command  CommandStrings
	Search   SearchStrings
	Links    LinksStrings
	Feeds    FeedsStrings
	Focus    FocusStrings
	Filters  FiltersStrings
	Errors   ErrorsStrings
	Common   CommonStrings
	Sort     SortStrings
	Catalog  CatalogStrings
	Library  LibraryStrings
}

// LibraryStrings covers the synthetic "Library" feed for arbitrary
// saved URLs (hotkey B): left-panel section label, modal prompts, and
// status/toast text for the background fetch.
type LibraryStrings struct {
	SectionLabel string // left-panel section name
	AddURLTitle  string // modal title
	AddURLPrompt string // input prompt label
	AddURLHint   string // footer hint inside the modal
	Saved        string // toast on successful save
	Fetching     string // status while fetching content
	FetchFailed  string // toast on fetch failure
	InvalidURL   string // toast / error when input is not a valid URL
	Deleted      string // toast after deleting a saved URL
}

type CatalogStrings struct {
	Title    string
	Subtitle string
	Hint     string
	Added    string
	Crumb    string

	// Onboarding welcome text.
	Welcome1 string // main greeting
	Welcome2 string // what rdr is
	Welcome3 string // what to do now

	// Default smart folder names (created on onboarding).
	FolderInbox    string
	FolderToday    string
	FolderThisWeek string
	FolderStarred   string
	FolderReadLater string
}

type SettingsStrings struct {
	Title               string
	SectionFeeds        string
	SectionGeneral      string
	SectionFolders      string
	SectionSmartFolders string
	SectionAfterSync    string
	SectionAI           string

	NewFeedName      string
	NewFeedURL       string
	RenameFeed       string
	CategoryPrompt   string
	ImportPrompt     string
	ExportPrompt     string
	EnterContinue    string
	EnterSave        string
	EnterSaveOrEmpty string
	NoFeeds          string
	FeedsHint        string

	// Regular folders (= categories on feeds).
	NoFolders    string
	FoldersHint  string
	FolderRename string

	// Category picker (opened via 'c' on a feed).
	NoFolderOption      string
	NewFolderOption     string
	CategoryPickerTitle string
	CategoryPickerHint  string

	// Smart folders (saved queries).
	NoSmartFolders       string
	SmartFoldersHint     string
	SmartFolderAddName   string
	SmartFolderAddQuery  string
	SmartFolderEditName  string
	SmartFolderEditQuery string

	GeneralHint   string
	LanguageLabel string
	ImagesLabel   string
	SortLabel     string
	PreviewLabel  string
	ThemeLabel    string
	RefreshLabel  string
	RefreshOff    string
	RefreshFmt    string // "%d min"

	AfterSyncTitle string
	AfterSyncHint  string
	AfterSyncAdd   string
	AfterSyncEdit  string
	NoAfterSync    string

	AIProviderLabel string
	AIEndpointLabel string
	AIKeyLabel      string
	AIModelLabel    string
	AIHint          string
	AINotConfigured string
}

type CommonStrings struct {
	On  string
	Off string
}

type SortStrings struct {
	DateDesc  string
	DateAsc   string
	TitleAsc  string
	TitleDesc string
}

type HelpStrings struct {
	TitleFmt        string // "rdr · help · %s"
	Footer          string
	SectionGlobal   string
	SectionNav      string
	SectionArticle  string
	SectionFilters  string
	SectionReader   string
	SectionFeedSet        string
	SectionFoldersSet     string
	SectionSmartFolderSet string
	SectionSearch         string
	SectionQuerySyn string
	SectionCommand  string
	SectionCommon   string
	SectionLinks    string
	SectionHelp     string

	DescDownUp          string
	DescSwitchPane      string
	DescOpenSelection   string
	DescTopBottom       string
	DescPagePrefix      string
	DescNextUnread      string
	DescCollapseCat     string
	DescToggleMarkAll   string
	DescToggleStar      string
	DescYankURLMD       string
	DescOpenBrowser     string
	DescShowAll         string
	DescShowUnread      string
	DescShowStarred     string
	DescScrollLine      string
	DescPageDown        string
	DescNextPrevArt     string
	DescBackToArticles  string
	DescLoadFull        string
	DescLinkPicker      string
	DescToggleRead      string
	DescToggleImages    string
	DescTogglePreview   string
	DescFeedsDownUp     string
	DescFeedsAdd        string
	DescFeedsDel        string
	DescFeedsRename     string
	DescFeedsSwitchSec  string
	DescFeedsClose      string
	DescFeedsImport     string
	DescFeedsExport     string
	DescFeedsCategory   string
	DescFoldersEdit     string
	DescFoldersDel      string
	DescSmartFolderAdd  string
	DescSmartFolderDel  string
	DescSmartFolderEdit string
	DescSearchNav       string
	DescSearchOpen      string
	DescSearchClose     string
	DescSynWord         string
	DescSynTitle        string
	DescSynFeed         string
	DescSynUnread       string
	DescSynStarred      string
	DescSynToday        string
	DescSynNewer        string
	DescSynNegate       string
	DescCmdNavSugg      string
	DescCmdComplete     string
	DescCmdHistory      string
	DescCmdExecute      string
	DescCmdCancel       string
	DescCmdSync         string
	DescCmdSort         string
	DescCmdFilter       string
	DescCmdReadQ        string
	DescCmdStarQ        string
	DescCmdCopyUrlQ     string
	DescCmdImportExport string
	DescCmdCollapseAll  string
	DescLinksNav        string
	DescLinksTop        string
	DescLinksOpen       string
	DescLinksClose      string
	DescHelpClose       string
	DescGlobalSearch    string
	DescGlobalCommand   string
	DescGlobalSync      string
	DescGlobalSettings  string
	DescGlobalZen       string
	DescGlobalHelp      string
	DescGlobalQuit      string
	DescGlobalAddURL    string
	DescDeleteLibrary   string
}

type KeysStrings struct {
	Quit          string
	Up            string
	Down          string
	Back          string
	Forward       string
	SwitchPane    string
	Top           string
	Bottom        string
	PageUp        string
	PageDown      string
	Open          string
	Esc           string
	RefreshOne    string
	RefreshAll    string
	OpenBrowser   string
	FullArticle   string
	Settings      string
	Search        string
	Help          string
	NextUnread    string
	Zen           string
	Command       string
	ToggleStar     string
	ToggleBookmark string
	FilterAll     string
	FilterUnread  string
	FilterStarred string
	NextArticle   string
	PrevArticle   string
	LinkPicker    string
	ToggleRead    string
	MarkAllRead   string
	YankURL       string
	YankMarkdown  string
	ToggleFold    string
	AddLibrary    string
	DeleteLibrary string
}

type ToastsStrings struct {
	Starred          string
	Unstarred        string
	Bookmarked       string
	Unbookmarked     string
	MarkedRead       string
	MarkedUnread     string
	MarkedReadFmt    string // "marked %d read"
	URLCopied        string
	MarkdownCopied   string
	CopiedFmt        string // "copied %d %s"
	BatchFmt         string // "%s · %d articles"
	ShowingFmt       string // "showing %s"
	LanguageChanged  string
	ThemeChangedFmt  string // "theme: %s"
	OpenedFmt        string // "opened: %s"
	SyncedOk         string // "synced %d feeds · %d new articles"
	SyncedOkNothing  string // "synced %d feeds · nothing new"
	SyncedErr        string // "synced %d feeds · %d errors"
	SyncedOkErrFmt   string // "synced %d feeds · %d new · %d errors"
	NoLinksInArticle string
}

type StatusStrings struct {
	Fetching           string
	AutoFetching       string
	Syncing            string
	SyncingNFmt        string // "syncing %d feeds…"
	LoadingFullArticle string
	MarkingRead        string
	MarkingUnread      string
	Starring           string
	Unstarring         string
	Copying            string
	Ready              string
	FullArticle        string
	EndOfList          string
	NextUnread         string
	NextFeedFmt        string // "next feed: %s"
	NoUnread           string
	SortFmt            string // "sort: %s"
	SortReversed       string
	CategoriesClosed   string
	CategoriesOpened   string
	ImagesOn           string
	ImagesOff          string
	PreviewOn          string
	PreviewOff         string
	ImportedFmt        string // "imported %d feeds"
	ExportedFmt        string // "exported %d feeds to %s"
	AppLabel           string // "rdr"
	SettingsCrumb      string // "rdr · settings"
	SearchCrumb        string
	LinksCrumb         string
	HelpCrumbFmt       string // "rdr · help · %s"
	StatusFmt          string // "rdr · %s"
	TinyPrefix         string // "rdr — %s"
	TooSmall           string // "rdr: terminal too small"
	SortBadge          string // "sort:%s%s"
	ZenBadge           string
	CountBadgeFmt      string // "count:%d"
	FilterBadgeFmt     string // "[%s]"
}

type ReaderStrings struct {
	EmptyHeadline   string
	PressF          string // "Press [f]"
	PressFSuffix    string // " to fetch & render it"
	PressFSuffix2   string // "with the readability extractor."
	LoadFullSuffix  string // " to load the full article"
	MinReadFmt      string // "%d min read"
	TimeAgoMinFmt   string // "%dm ago"
	TimeAgoHourFmt  string // "%dh ago"
	TimeAgoDayFmt   string // "%dd ago"
	BucketToday     string
	BucketYesterday string
	BucketWeek      string
	BucketMonth     string
	BucketOlder     string

	Translating string
	Summarizing string
	Translated  string
	Summarized  string
}

type CommandStrings struct {
	NoMatching   string
	MoreFmt      string // "  … +%d more"
	HelpSync     string
	HelpRefresh  string
	HelpSortDate string
	HelpSortTitle string
	HelpSortReverse string
	HelpFilterAll string
	HelpFilterUnread string
	HelpFilterStarred string
	HelpStar       string
	HelpBookmark   string
	HelpUnbookmark string
	HelpRead     string
	HelpUnread   string
	HelpUnstar   string
	HelpCopyURL  string
	HelpCopyMD   string
	HelpImport   string
	HelpExport   string
	HelpImages   string
	HelpCollapseAll string
	HelpExpandAll string
	HelpZen      string
	HelpHelp     string
	HelpDiscover string
	HelpSettings string
	HelpSearch   string
	HelpQuit     string
	HelpQuitAlias string
}

type SearchStrings struct {
	Title           string
	PreviewTitle    string
	NoArticles      string
	NoArticlesHint  string
	NoMatches       string
	NoMatchesHint   string
	NoSelection     string
	ResultsFmt      string // "%d/%d results"
	NoPreviewHint   string
	SyntaxHint      string
}

type LinksStrings struct {
	TitleFmt  string // "Links · %d"
	NoLinks   string
	FooterHint string
}

type FeedsStrings struct {
	PaneTitle         string
	NoFeeds           string
	OtherCategory     string
	ArticlesPaneTitle string
	NoArticles        string
	NoArticlePreview  string
}

type FocusStrings struct {
	Feeds    string
	Articles string
	Reader   string
	Settings string
	Search   string
	Command  string
	Links    string
	Help     string
	AddURL   string
}

type FiltersStrings struct {
	All     string
	Unread  string
	Starred string
}

type ErrorsStrings struct {
	SortNeedsArg      string
	UnknownSortFmt    string
	FilterNeedsArg    string
	UnknownFilterFmt  string
	ReadNeedsQuery    string
	UnreadNeedsQuery  string
	UnstarNeedsQuery  string
	CopyNeedsArg      string
	UnknownCopyFmt    string
	ImportNeedsPath   string
	ExportNeedsPath   string
	UnknownCommandFmt string
	NoURLToCopy       string
	NoMatches         string
}

var en = Strings{
	Settings: SettingsStrings{
		Title:               "Settings",
		SectionFeeds:        "Feeds",
		SectionGeneral:      "General",
		SectionFolders:      "Folders",
		SectionSmartFolders: "Smart folders",
		SectionAfterSync:    "Auto-commands",
		SectionAI:           "AI",

		NewFeedName:      "New feed name:",
		NewFeedURL:       "New feed URL:",
		RenameFeed:       "Rename feed:",
		CategoryPrompt:   "Category:",
		ImportPrompt:     "Import OPML from:",
		ExportPrompt:     "Export OPML to:",
		EnterContinue:    "enter to continue · esc to cancel",
		EnterSave:        "enter to save · esc to cancel",
		EnterSaveOrEmpty: "enter to save (empty = Other) · esc to cancel",
		NoFeeds:          "(no feeds) — press a to add",
		FeedsHint:        "a add · d del · e rename · c cat · i import · E export · tab section · esc close",

		NoFolders:    "(no folders)",
		FoldersHint:  "e rename · d delete · tab section · esc close",
		FolderRename: "Rename folder:",

		NoFolderOption:      "— (no folder)",
		NewFolderOption:     "+ New folder…",
		CategoryPickerTitle: "Move feed to folder:",
		CategoryPickerHint:  "j/k select · enter apply · esc cancel",

		NoSmartFolders:       "(no smart folders) — press a to add",
		SmartFoldersHint:     "a add · d del · e edit · tab section · esc close",
		SmartFolderAddName:   "New smart folder name:",
		SmartFolderAddQuery:  "Smart folder query:",
		SmartFolderEditName:  "Rename smart folder:",
		SmartFolderEditQuery: "Edit smart folder query:",

		GeneralHint:   "j/k select · enter/space toggle · tab section · esc close",
		LanguageLabel: "Language",
		ImagesLabel:   "Images",
		SortLabel:     "Sort",
		PreviewLabel:  "Preview",
		ThemeLabel:    "Theme",
		RefreshLabel:  "Auto-refresh",
		RefreshOff:    "disabled",
		RefreshFmt:    "%d min",

		AfterSyncTitle: "After-sync commands",
		AfterSyncHint:  "a add · d delete · e edit · esc close",
		AfterSyncAdd:   "New command (query syntax):",
		AfterSyncEdit:  "Edit command:",
		NoAfterSync:    "No after-sync commands",

		AIProviderLabel: "Provider",
		AIEndpointLabel: "Endpoint",
		AIKeyLabel:      "API Key",
		AIModelLabel:    "Model",
		AIHint:          "enter toggle/edit · tab section · esc close",
		AINotConfigured: "Not configured — choose provider and set up to enable translation and summarization",
	},
	Help: HelpStrings{
		TitleFmt:        "rdr · help · %s",
		Footer:          "esc close · ? toggle",
		SectionGlobal:   "Global",
		SectionNav:      "Navigation",
		SectionArticle:  "Article ops",
		SectionFilters:  "Filters",
		SectionReader:   "Reader",
		SectionFeedSet:        "Feed settings",
		SectionFoldersSet:     "Folder settings",
		SectionSmartFolderSet: "Smart folder settings",
		SectionSearch:         "Search picker",
		SectionQuerySyn: "Query syntax",
		SectionCommand:  "Command mode",
		SectionCommon:   "Common commands",
		SectionLinks:    "Link picker",
		SectionHelp:     "Help",

		DescDownUp:          "down / up",
		DescSwitchPane:      "switch pane",
		DescOpenSelection:   "open selection",
		DescTopBottom:       "top / bottom",
		DescPagePrefix:      "page up / down",
		DescNextUnread:      "next unread",
		DescCollapseCat:     "collapse category (feeds pane)",
		DescToggleMarkAll:   "toggle / mark all read",
		DescToggleStar:      "toggle star",
		DescYankURLMD:       "yank URL / [title](url)",
		DescOpenBrowser:     "open in browser",
		DescShowAll:         "show all articles",
		DescShowUnread:      "show unread only",
		DescShowStarred:     "show starred only",
		DescScrollLine:      "scroll line",
		DescPageDown:        "page down",
		DescNextPrevArt:     "next / prev article",
		DescBackToArticles:  "back to articles",
		DescLoadFull:        "load full article",
		DescLinkPicker:      "link picker",
		DescToggleRead:      "toggle read",
		DescToggleImages:    "toggle inline images",
		DescTogglePreview:   "toggle preview popup",
		DescFeedsDownUp:     "down / up",
		DescFeedsAdd:        "add feed",
		DescFeedsDel:        "delete feed",
		DescFeedsRename:     "rename feed",
		DescFeedsSwitchSec:  "switch section",
		DescFeedsClose:      "close",
		DescFeedsImport:     "import OPML",
		DescFeedsExport:     "export OPML",
		DescFeedsCategory:   "set feed folder",
		DescFoldersEdit:     "rename folder",
		DescFoldersDel:      "delete folder",
		DescSmartFolderAdd:  "add smart folder",
		DescSmartFolderDel:  "delete smart folder",
		DescSmartFolderEdit: "edit smart folder",
		DescSearchNav:       "navigate results",
		DescSearchOpen:      "open selection",
		DescSearchClose:     "close",
		DescSynWord:         "match title or feed name",
		DescSynTitle:        "field match",
		DescSynFeed:         "feed name",
		DescSynUnread:       "only unread",
		DescSynStarred:      "only starred",
		DescSynToday:        "published today",
		DescSynNewer:        "newer than 1 week",
		DescSynNegate:       "negate any atom",
		DescCmdNavSugg:      "navigate suggestions",
		DescCmdComplete:     "complete highlighted",
		DescCmdHistory:      "history prev / next",
		DescCmdExecute:      "execute",
		DescCmdCancel:       "cancel",
		DescCmdSync:         "refresh all feeds",
		DescCmdSort:         "change sort",
		DescCmdFilter:       "set filter",
		DescCmdReadQ:        "mark matches read",
		DescCmdStarQ:        "star matches",
		DescCmdCopyUrlQ:     "copy URLs to clipboard",
		DescCmdImportExport: "OPML in/out",
		DescCmdCollapseAll:  "toggle all categories",
		DescLinksNav:        "navigate",
		DescLinksTop:        "top / bottom",
		DescLinksOpen:       "open in browser",
		DescLinksClose:      "close",
		DescHelpClose:       "close",
		DescGlobalSearch:    "open search picker",
		DescGlobalCommand:   "open command mode",
		DescGlobalSync:      "sync all feeds",
		DescGlobalSettings:  "open settings",
		DescGlobalZen:       "toggle zen mode",
		DescGlobalHelp:      "toggle this help",
		DescGlobalQuit:      "quit",
		DescGlobalAddURL:    "save URL to Library",
		DescDeleteLibrary:   "delete from Library",
	},
	Keys: KeysStrings{
		Quit:          "quit",
		Up:            "up",
		Down:          "down",
		Back:          "back",
		Forward:       "forward",
		SwitchPane:    "switch pane",
		Top:           "top",
		Bottom:        "bottom",
		PageUp:        "page up",
		PageDown:      "page down",
		Open:          "open",
		Esc:           "back",
		RefreshOne:    "refresh current",
		RefreshAll:    "refresh all",
		OpenBrowser:   "open in browser",
		FullArticle:   "full article",
		Settings:      "settings",
		Search:        "search",
		Help:          "help",
		NextUnread:    "next unread",
		Zen:           "zen",
		Command:       "command",
		ToggleStar:     "toggle star",
		ToggleBookmark: "read later",
		FilterAll:      "all articles",
		FilterUnread:  "unread only",
		FilterStarred: "starred only",
		NextArticle:   "next article",
		PrevArticle:   "prev article",
		LinkPicker:    "links",
		ToggleRead:    "toggle read",
		MarkAllRead:   "mark all read",
		YankURL:       "yank URL",
		YankMarkdown:  "yank [title](url)",
		ToggleFold:    "collapse category",
		AddLibrary:    "save URL",
		DeleteLibrary: "delete from Library",
	},
	Toasts: ToastsStrings{
		Starred:          "★ starred",
		Unstarred:        "unstarred",
		Bookmarked:       "saved for later",
		Unbookmarked:     "removed from read later",
		MarkedRead:       "marked read",
		MarkedUnread:     "marked unread",
		MarkedReadFmt:    "marked %d read",
		URLCopied:        "URL copied",
		MarkdownCopied:   "markdown copied",
		CopiedFmt:        "copied %d %s",
		BatchFmt:         "%s · %d articles",
		ShowingFmt:       "showing %s",
		LanguageChanged:  "language changed",
		ThemeChangedFmt:  "theme: %s",
		OpenedFmt:        "opened: %s",
		SyncedOk:         "synced %d feeds · %d new articles",
		SyncedOkNothing:  "synced %d feeds · nothing new",
		SyncedErr:        "synced %d feeds · %d error(s)",
		SyncedOkErrFmt:   "synced %d feeds · %d new · %d error(s)",
		NoLinksInArticle: "no links in article",
	},
	Status: StatusStrings{
		Fetching:           "fetching…",
		AutoFetching:       "auto-fetching…",
		Syncing:            "syncing…",
		SyncingNFmt:        "syncing %d feeds…",
		LoadingFullArticle: "loading full article…",
		MarkingRead:        "marking read…",
		MarkingUnread:      "marking unread…",
		Starring:           "starring…",
		Unstarring:         "unstarring…",
		Copying:            "copying…",
		Ready:              "ready",
		FullArticle:        "full article",
		EndOfList:          "end of list",
		NextUnread:         "next unread",
		NextFeedFmt:        "next feed: %s",
		NoUnread:           "no unread",
		SortFmt:            "sort: %s",
		SortReversed:       "sort reversed",
		CategoriesClosed:   "categories collapsed",
		CategoriesOpened:   "categories expanded",
		ImagesOn:           "images on",
		ImagesOff:          "images off",
		PreviewOn:          "preview on",
		PreviewOff:         "preview off",
		ImportedFmt:        "imported %d feeds",
		ExportedFmt:        "exported %d feeds to %s",
		AppLabel:           "rdr",
		SettingsCrumb:      "rdr · settings",
		SearchCrumb:        "rdr · search",
		LinksCrumb:         "rdr · links",
		HelpCrumbFmt:       "rdr · help · %s",
		StatusFmt:          "rdr · %s",
		TinyPrefix:         "rdr — %s",
		TooSmall:           "rdr: terminal too small",
		SortBadge:          "sort:%s%s",
		ZenBadge:           "zen",
		CountBadgeFmt:      "count:%d",
		FilterBadgeFmt:     "[%s]",
	},
	Reader: ReaderStrings{
		EmptyHeadline:   "This article has no full body loaded yet.",
		PressF:          "Press [f]",
		PressFSuffix:    " to fetch & render it",
		PressFSuffix2:   "with the readability extractor.",
		LoadFullSuffix:  " to load the full article",
		MinReadFmt:      "%d min read",
		TimeAgoMinFmt:   "%dm ago",
		TimeAgoHourFmt:  "%dh ago",
		TimeAgoDayFmt:   "%dd ago",
		BucketToday:     "Today",
		BucketYesterday: "Yesterday",
		BucketWeek:      "This week",
		BucketMonth:     "This month",
		BucketOlder:     "Older",
		Translating:     "translating…",
		Summarizing:     "summarizing…",
		Translated:      "translation ready",
		Summarized:      "summary ready",
	},
	Command: CommandStrings{
		NoMatching:        "(no matching commands)",
		MoreFmt:           "  … +%d more",
		HelpSync:          "Fetch all feeds",
		HelpRefresh:       "Fetch all feeds (alias for sync)",
		HelpSortDate:      "Sort articles by publish date",
		HelpSortTitle:     "Sort articles alphabetically",
		HelpSortReverse:   "Toggle sort direction",
		HelpFilterAll:     "Show all articles",
		HelpFilterUnread:  "Show only unread articles",
		HelpFilterStarred: "Show only starred articles",
		HelpStar:          "Toggle star on current article",
		HelpBookmark:      "Toggle read later (:bookmark <query>)",
		HelpUnbookmark:    "Remove from read later (:unbookmark <query>)",
		HelpRead:          "Mark matching articles read (:read <query>)",
		HelpUnread:        "Mark matching articles unread (:unread <query>)",
		HelpUnstar:        "Unstar matching articles (:unstar <query>)",
		HelpCopyURL:       "Copy matching URLs to clipboard (:copy url <query>)",
		HelpCopyMD:        "Copy matches as markdown list (:copy md <query>)",
		HelpImport:        "Import feeds from OPML file (:import <path>)",
		HelpExport:        "Export feeds to OPML file (:export <path>)",
		HelpImages:        "Toggle image markdown in reader",
		HelpCollapseAll:   "Collapse all feed categories",
		HelpExpandAll:     "Expand all feed categories",
		HelpZen:           "Toggle zen mode",
		HelpHelp:          "Toggle help overlay",
		HelpDiscover:      "Browse feed catalog",
		HelpSettings:      "Open feed settings",
		HelpSearch:        "Open search picker",
		HelpQuit:          "Exit rdr",
		HelpQuitAlias:     "Exit rdr (alias for quit)",
	},
	Search: SearchStrings{
		Title:          "Search",
		PreviewTitle:   "Preview",
		NoArticles:     "No articles yet",
		NoArticlesHint: "Sync feeds first — press R to refresh all",
		NoMatches:      "No matches",
		NoMatchesHint:  "Try a broader query or clear the input (ctrl+u)",
		NoSelection:    "(no selection)",
		ResultsFmt:     "%d/%d results",
		NoPreviewHint:  "(no preview — press enter, then f to load full)",
		SyntaxHint:     "title:foo · feed:bar · unread · starred · newer:1w · ~negate · ? for help",
	},
	Links: LinksStrings{
		TitleFmt:   "Links · %d",
		NoLinks:    "(no links)",
		FooterHint: "↑↓ navigate · enter open · esc back",
	},
	Feeds: FeedsStrings{
		PaneTitle:         "Feeds",
		NoFeeds:           "(no feeds)",
		OtherCategory:     "Other",
		ArticlesPaneTitle: "Articles",
		NoArticles:        "(no articles)",
		NoArticlePreview:  "(no preview available)",
	},
	Focus: FocusStrings{
		Feeds:    "Feeds",
		Articles: "Articles",
		Reader:   "Reader",
		Settings: "Settings",
		Search:   "Search",
		Command:  "Command",
		Links:    "Links",
		Help:     "Help",
		AddURL:   "Save URL",
	},
	Filters: FiltersStrings{
		All:     "all",
		Unread:  "unread",
		Starred: "starred",
	},
	Errors: ErrorsStrings{
		SortNeedsArg:      ":sort needs date|title",
		UnknownSortFmt:    "unknown sort field %q",
		FilterNeedsArg:    ":filter needs all|unread|starred",
		UnknownFilterFmt:  "unknown filter %q",
		ReadNeedsQuery:    ":read needs a query",
		UnreadNeedsQuery:  ":unread needs a query",
		UnstarNeedsQuery:  ":unstar needs a query",
		CopyNeedsArg:      ":copy needs: url|md <query>",
		UnknownCopyFmt:    "unknown copy format %q, expected url|md",
		ImportNeedsPath:   ":import needs a path",
		ExportNeedsPath:   ":export needs a path",
		UnknownCommandFmt: "unknown command %q",
		NoURLToCopy:       "no URL to copy",
		NoMatches:         "no matches",
	},
	Common: CommonStrings{
		On:  "on",
		Off: "off",
	},
	Sort: SortStrings{
		DateDesc:  "date ↓",
		DateAsc:   "date ↑",
		TitleAsc:  "title ↑",
		TitleDesc: "title ↓",
	},
	Catalog: CatalogStrings{
		Title:    "Discover Feeds",
		Subtitle: "Browse curated RSS feeds and subscribe with Enter",
		Hint:     "j/k navigate · enter subscribe · esc close",
		Added:    "subscribed",
		Crumb:    "discover",

		Welcome1: "Welcome to rdr!",
		Welcome2: "A terminal RSS reader with vim-style navigation, full article rendering, smart folders, and search with a query language.",
		Welcome3: "Pick a few feeds below to get started. Press Enter to subscribe, then Esc to start reading.",

		FolderInbox:    "Inbox",
		FolderToday:    "Today",
		FolderThisWeek: "This Week",
		FolderStarred:   "Starred",
		FolderReadLater: "Read Later",
	},
	Library: LibraryStrings{
		SectionLabel: "Library",
		AddURLTitle:  "Save URL to Library",
		AddURLPrompt: "URL:",
		AddURLHint:   "enter to save · esc to cancel",
		Saved:        "saved to Library",
		Fetching:     "fetching saved URL…",
		FetchFailed:  "failed to fetch URL",
		InvalidURL:   "invalid URL",
		Deleted:      "deleted from Library",
	},
}

var ru = Strings{
	Settings: SettingsStrings{
		Title:               "Настройки",
		SectionFeeds:        "Ленты",
		SectionGeneral:      "Общие",
		SectionFolders:      "Папки",
		SectionSmartFolders: "Умные папки",
		SectionAfterSync:    "Автокоманды",
		SectionAI:           "AI",

		NewFeedName:      "Название ленты:",
		NewFeedURL:       "URL ленты:",
		RenameFeed:       "Переименовать ленту:",
		CategoryPrompt:   "Папка ленты:",
		ImportPrompt:     "Импорт OPML из:",
		ExportPrompt:     "Экспорт OPML в:",
		EnterContinue:    "enter продолжить · esc отмена",
		EnterSave:        "enter сохранить · esc отмена",
		EnterSaveOrEmpty: "enter сохранить (пусто = Прочее) · esc отмена",
		NoFeeds:          "(лент нет) — нажмите a чтобы добавить",
		FeedsHint:        "a доб · d удал · e переим · c папка · i импорт · E экспорт · tab раздел · esc закрыть",

		NoFolders:    "(папок нет)",
		FoldersHint:  "e переим · d удалить · tab раздел · esc закрыть",
		FolderRename: "Переименовать папку:",

		NoFolderOption:      "— (без папки)",
		NewFolderOption:     "+ Новая папка…",
		CategoryPickerTitle: "Переместить ленту в папку:",
		CategoryPickerHint:  "j/k выбрать · enter применить · esc отмена",

		NoSmartFolders:       "(умных папок нет) — нажмите a чтобы добавить",
		SmartFoldersHint:     "a добавить · d удалить · e править · tab раздел · esc закрыть",
		SmartFolderAddName:   "Название умной папки:",
		SmartFolderAddQuery:  "Запрос умной папки:",
		SmartFolderEditName:  "Переименовать умную папку:",
		SmartFolderEditQuery: "Изменить запрос:",

		GeneralHint:   "j/k выбрать · enter/space переключить · tab раздел · esc закрыть",
		LanguageLabel: "Язык",
		ImagesLabel:   "Картинки",
		SortLabel:     "Сортировка",
		PreviewLabel:  "Превью",
		ThemeLabel:    "Тема",
		RefreshLabel:  "Автообновление",
		RefreshOff:    "отключено",
		RefreshFmt:    "%d мин",

		AfterSyncTitle: "Автокоманды после синхронизации",
		AfterSyncHint:  "a добавить · d удалить · e править · esc закрыть",
		AfterSyncAdd:   "Новая команда (синтаксис запросов):",
		AfterSyncEdit:  "Редактировать команду:",
		NoAfterSync:    "Нет автокоманд",

		AIProviderLabel: "Провайдер",
		AIEndpointLabel: "Endpoint",
		AIKeyLabel:      "API Key",
		AIModelLabel:    "Модель",
		AIHint:          "enter переключить/править · tab раздел · esc закрыть",
		AINotConfigured: "Не настроен — выберите провайдер и настройте для перевода и суммаризации",
	},
	Help: HelpStrings{
		TitleFmt:        "rdr · справка · %s",
		Footer:          "esc закрыть · ? переключить",
		SectionGlobal:   "Общее",
		SectionNav:      "Навигация",
		SectionArticle:  "Статья",
		SectionFilters:  "Фильтры",
		SectionReader:   "Читалка",
		SectionFeedSet:        "Настройки лент",
		SectionFoldersSet:     "Настройки папок",
		SectionSmartFolderSet: "Настройки умных папок",
		SectionSearch:         "Поиск",
		SectionQuerySyn: "Синтаксис запросов",
		SectionCommand:  "Командная строка",
		SectionCommon:   "Частые команды",
		SectionLinks:    "Выбор ссылок",
		SectionHelp:     "Справка",

		DescDownUp:          "вниз / вверх",
		DescSwitchPane:      "переключить панель",
		DescOpenSelection:   "открыть выделенное",
		DescTopBottom:       "в начало / в конец",
		DescPagePrefix:      "страница вверх / вниз",
		DescNextUnread:      "следующая непрочитанная",
		DescCollapseCat:     "свернуть категорию (панель лент)",
		DescToggleMarkAll:   "переключить / отметить все прочитанными",
		DescToggleStar:      "переключить звезду",
		DescYankURLMD:       "копировать URL / [title](url)",
		DescOpenBrowser:     "открыть в браузере",
		DescShowAll:         "показать все статьи",
		DescShowUnread:      "только непрочитанные",
		DescShowStarred:     "только со звездой",
		DescScrollLine:      "прокрутить строку",
		DescPageDown:        "страница вниз",
		DescNextPrevArt:     "следующая / предыдущая статья",
		DescBackToArticles:  "назад к статьям",
		DescLoadFull:        "загрузить полную статью",
		DescLinkPicker:      "выбор ссылок",
		DescToggleRead:      "переключить прочитано",
		DescToggleImages:    "переключить картинки",
		DescTogglePreview:   "переключить окно превью",
		DescFeedsDownUp:     "вниз / вверх",
		DescFeedsAdd:        "добавить ленту",
		DescFeedsDel:        "удалить ленту",
		DescFeedsRename:     "переименовать ленту",
		DescFeedsSwitchSec:  "переключить раздел",
		DescFeedsClose:      "закрыть",
		DescFeedsImport:     "импорт OPML",
		DescFeedsExport:     "экспорт OPML",
		DescFeedsCategory:   "задать папку ленты",
		DescFoldersEdit:     "переименовать папку",
		DescFoldersDel:      "удалить папку",
		DescSmartFolderAdd:  "добавить умную папку",
		DescSmartFolderDel:  "удалить умную папку",
		DescSmartFolderEdit: "изменить умную папку",
		DescSearchNav:       "навигация по результатам",
		DescSearchOpen:      "открыть выделенное",
		DescSearchClose:     "закрыть",
		DescSynWord:         "совпадение в заголовке или имени ленты",
		DescSynTitle:        "совпадение в поле",
		DescSynFeed:         "имя ленты",
		DescSynUnread:       "только непрочитанные",
		DescSynStarred:      "только со звездой",
		DescSynToday:        "опубликовано сегодня",
		DescSynNewer:        "новее 1 недели",
		DescSynNegate:       "инвертировать атом",
		DescCmdNavSugg:      "навигация по подсказкам",
		DescCmdComplete:     "подставить выделенное",
		DescCmdHistory:      "история: назад / вперёд",
		DescCmdExecute:      "выполнить",
		DescCmdCancel:       "отмена",
		DescCmdSync:         "обновить все ленты",
		DescCmdSort:         "сменить сортировку",
		DescCmdFilter:       "установить фильтр",
		DescCmdReadQ:        "отметить совпадения прочитанными",
		DescCmdStarQ:        "звезда на совпадения",
		DescCmdCopyUrlQ:     "копировать URL в буфер",
		DescCmdImportExport: "OPML импорт/экспорт",
		DescCmdCollapseAll:  "свернуть / развернуть все категории",
		DescLinksNav:        "навигация",
		DescLinksTop:        "в начало / в конец",
		DescLinksOpen:       "открыть в браузере",
		DescLinksClose:      "закрыть",
		DescHelpClose:       "закрыть",
		DescGlobalSearch:    "открыть поиск",
		DescGlobalCommand:   "открыть командный режим",
		DescGlobalSync:      "синхронизировать все ленты",
		DescGlobalSettings:  "открыть настройки",
		DescGlobalZen:       "переключить zen-режим",
		DescGlobalHelp:      "переключить справку",
		DescGlobalQuit:      "выход",
		DescGlobalAddURL:    "сохранить URL в библиотеку",
		DescDeleteLibrary:   "удалить из библиотеки",
	},
	Keys: KeysStrings{
		Quit:          "выход",
		Up:            "вверх",
		Down:          "вниз",
		Back:          "назад",
		Forward:       "вперёд",
		SwitchPane:    "переключить панель",
		Top:           "в начало",
		Bottom:        "в конец",
		PageUp:        "страница вверх",
		PageDown:      "страница вниз",
		Open:          "открыть",
		Esc:           "назад",
		RefreshOne:    "обновить текущую",
		RefreshAll:    "обновить все",
		OpenBrowser:   "открыть в браузере",
		FullArticle:   "полная статья",
		Settings:      "настройки",
		Search:        "поиск",
		Help:          "справка",
		NextUnread:    "следующая непрочитанная",
		Zen:           "zen",
		Command:       "команда",
		ToggleStar:     "звезда",
		ToggleBookmark: "почитать позже",
		FilterAll:      "все статьи",
		FilterUnread:  "непрочитанные",
		FilterStarred: "со звездой",
		NextArticle:   "следующая статья",
		PrevArticle:   "предыдущая статья",
		LinkPicker:    "ссылки",
		ToggleRead:    "прочитано",
		MarkAllRead:   "отметить все",
		YankURL:       "копировать URL",
		YankMarkdown:  "копировать [title](url)",
		ToggleFold:    "свернуть категорию",
		AddLibrary:    "сохранить URL",
		DeleteLibrary: "удалить из библиотеки",
	},
	Toasts: ToastsStrings{
		Starred:          "★ со звездой",
		Unstarred:        "звезда снята",
		Bookmarked:       "добавлено в «почитать позже»",
		Unbookmarked:     "убрано из «почитать позже»",
		MarkedRead:       "прочитано",
		MarkedUnread:     "непрочитано",
		MarkedReadFmt:    "прочитано: %d",
		URLCopied:        "URL скопирован",
		MarkdownCopied:   "markdown скопирован",
		CopiedFmt:        "скопировано %d %s",
		BatchFmt:         "%s · статей: %d",
		ShowingFmt:       "показываю: %s",
		LanguageChanged:  "язык изменён",
		ThemeChangedFmt:  "тема: %s",
		OpenedFmt:        "открыто: %s",
		SyncedOk:         "синхронизация: лент %d · новых статей %d",
		SyncedOkNothing:  "синхронизация: лент %d · ничего нового",
		SyncedErr:        "синхронизация: лент %d · ошибок %d",
		SyncedOkErrFmt:   "синхронизация: лент %d · новых %d · ошибок %d",
		NoLinksInArticle: "в статье нет ссылок",
	},
	Status: StatusStrings{
		Fetching:           "загрузка…",
		AutoFetching:       "автообновление…",
		Syncing:            "синхронизация…",
		SyncingNFmt:        "синхронизация лент: %d…",
		LoadingFullArticle: "загрузка полной статьи…",
		MarkingRead:        "отмечаю прочитанным…",
		MarkingUnread:      "отмечаю непрочитанным…",
		Starring:           "ставлю звезду…",
		Unstarring:         "снимаю звезду…",
		Copying:            "копирую…",
		Ready:              "готово",
		FullArticle:        "полная статья",
		EndOfList:          "конец списка",
		NextUnread:         "следующая непрочитанная",
		NextFeedFmt:        "следующая лента: %s",
		NoUnread:           "нет непрочитанных",
		SortFmt:            "сортировка: %s",
		SortReversed:       "порядок обращён",
		CategoriesClosed:   "категории свёрнуты",
		CategoriesOpened:   "категории развёрнуты",
		ImagesOn:           "картинки включены",
		ImagesOff:          "картинки выключены",
		PreviewOn:          "превью включено",
		PreviewOff:         "превью выключено",
		ImportedFmt:        "импортировано лент: %d",
		ExportedFmt:        "экспортировано лент: %d в %s",
		AppLabel:           "rdr",
		SettingsCrumb:      "rdr · настройки",
		SearchCrumb:        "rdr · поиск",
		LinksCrumb:         "rdr · ссылки",
		HelpCrumbFmt:       "rdr · справка · %s",
		StatusFmt:          "rdr · %s",
		TinyPrefix:         "rdr — %s",
		TooSmall:           "rdr: терминал слишком мал",
		SortBadge:          "sort:%s%s",
		ZenBadge:           "zen",
		CountBadgeFmt:      "count:%d",
		FilterBadgeFmt:     "[%s]",
	},
	Reader: ReaderStrings{
		EmptyHeadline:   "Полный текст статьи ещё не загружен.",
		PressF:          "Нажмите [f]",
		PressFSuffix:    " чтобы загрузить и отрендерить",
		PressFSuffix2:   "через экстрактор читабельности.",
		LoadFullSuffix:  " чтобы загрузить полную статью",
		MinReadFmt:      "%d мин чтения",
		TimeAgoMinFmt:   "%d мин назад",
		TimeAgoHourFmt:  "%d ч назад",
		TimeAgoDayFmt:   "%d дн назад",
		BucketToday:     "Сегодня",
		BucketYesterday: "Вчера",
		BucketWeek:      "На этой неделе",
		BucketMonth:     "В этом месяце",
		BucketOlder:     "Ранее",
		Translating:     "перевод…",
		Summarizing:     "суммаризация…",
		Translated:      "перевод готов",
		Summarized:      "сводка готова",
	},
	Command: CommandStrings{
		NoMatching:        "(совпадений нет)",
		MoreFmt:           "  … ещё +%d",
		HelpSync:          "Обновить все ленты",
		HelpRefresh:       "Обновить все ленты (псевдоним sync)",
		HelpSortDate:      "Сортировать по дате публикации",
		HelpSortTitle:     "Сортировать по алфавиту",
		HelpSortReverse:   "Переключить порядок сортировки",
		HelpFilterAll:     "Показать все статьи",
		HelpFilterUnread:  "Только непрочитанные",
		HelpFilterStarred: "Только со звездой",
		HelpStar:          "Переключить звезду на текущей статье",
		HelpBookmark:      "Почитать позже (:bookmark <запрос>)",
		HelpUnbookmark:    "Убрать из «почитать позже» (:unbookmark <запрос>)",
		HelpRead:          "Отметить совпадения прочитанными (:read <запрос>)",
		HelpUnread:        "Отметить совпадения непрочитанными (:unread <запрос>)",
		HelpUnstar:        "Снять звезду со совпадений (:unstar <запрос>)",
		HelpCopyURL:       "Копировать URL совпадений в буфер (:copy url <запрос>)",
		HelpCopyMD:        "Копировать совпадения как markdown-список (:copy md <запрос>)",
		HelpImport:        "Импортировать ленты из OPML (:import <путь>)",
		HelpExport:        "Экспортировать ленты в OPML (:export <путь>)",
		HelpImages:        "Переключить картинки в читалке",
		HelpCollapseAll:   "Свернуть все категории лент",
		HelpExpandAll:     "Развернуть все категории лент",
		HelpZen:           "Переключить zen-режим",
		HelpHelp:          "Переключить окно справки",
		HelpDiscover:      "Каталог RSS-лент",
		HelpSettings:      "Открыть настройки",
		HelpSearch:        "Открыть поиск",
		HelpQuit:          "Выйти из rdr",
		HelpQuitAlias:     "Выйти из rdr (псевдоним quit)",
	},
	Search: SearchStrings{
		Title:          "Поиск",
		PreviewTitle:   "Превью",
		NoArticles:     "Пока нет статей",
		NoArticlesHint: "Сначала синхронизируйте ленты — нажмите R",
		NoMatches:      "Ничего не найдено",
		NoMatchesHint:  "Попробуйте упростить запрос или очистите поле (ctrl+u)",
		NoSelection:    "(нет выбора)",
		ResultsFmt:     "результатов: %d/%d",
		NoPreviewHint:  "(нет превью — нажмите enter, затем f для полной)",
		SyntaxHint:     "title:foo · feed:bar · unread · starred · newer:1w · ~negate · ? для справки",
	},
	Links: LinksStrings{
		TitleFmt:   "Ссылки · %d",
		NoLinks:    "(ссылок нет)",
		FooterHint: "↑↓ навигация · enter открыть · esc назад",
	},
	Feeds: FeedsStrings{
		PaneTitle:         "Ленты",
		NoFeeds:           "(лент нет)",
		OtherCategory:     "Прочее",
		ArticlesPaneTitle: "Статьи",
		NoArticles:        "(статей нет)",
		NoArticlePreview:  "(нет превью)",
	},
	Focus: FocusStrings{
		Feeds:    "Ленты",
		Articles: "Статьи",
		Reader:   "Читалка",
		Settings: "Настройки",
		Search:   "Поиск",
		Command:  "Команда",
		Links:    "Ссылки",
		Help:     "Справка",
		AddURL:   "Сохранить URL",
	},
	Filters: FiltersStrings{
		All:     "все",
		Unread:  "непрочитанные",
		Starred: "со звездой",
	},
	Errors: ErrorsStrings{
		SortNeedsArg:      ":sort требует date|title",
		UnknownSortFmt:    "неизвестное поле сортировки %q",
		FilterNeedsArg:    ":filter требует all|unread|starred",
		UnknownFilterFmt:  "неизвестный фильтр %q",
		ReadNeedsQuery:    ":read требует запрос",
		UnreadNeedsQuery:  ":unread требует запрос",
		UnstarNeedsQuery:  ":unstar требует запрос",
		CopyNeedsArg:      ":copy требует: url|md <запрос>",
		UnknownCopyFmt:    "неизвестный формат копирования %q, ожидаю url|md",
		ImportNeedsPath:   ":import требует путь",
		ExportNeedsPath:   ":export требует путь",
		UnknownCommandFmt: "неизвестная команда %q",
		NoURLToCopy:       "нет URL для копирования",
		NoMatches:         "совпадений нет",
	},
	Common: CommonStrings{
		On:  "вкл",
		Off: "выкл",
	},
	Sort: SortStrings{
		DateDesc:  "дата ↓",
		DateAsc:   "дата ↑",
		TitleAsc:  "заголовок ↑",
		TitleDesc: "заголовок ↓",
	},
	Catalog: CatalogStrings{
		Title:    "Каталог лент",
		Subtitle: "Обзор RSS-лент по категориям. Enter — подписаться",
		Hint:     "j/k навигация · enter подписаться · esc закрыть",
		Added:    "подписан",
		Crumb:    "каталог",

		Welcome1: "Добро пожаловать в rdr!",
		Welcome2: "Терминальная читалка RSS с vim-навигацией, полнотекстовым чтением статей, умными папками и поиском с языком запросов.",
		Welcome3: "Выберите ленты ниже, чтобы начать. Enter — подписаться, затем Esc — перейти к чтению.",

		FolderInbox:    "Входящие",
		FolderToday:    "Сегодня",
		FolderThisWeek: "За неделю",
		FolderStarred:   "Избранные",
		FolderReadLater: "Почитать позже",
	},
	Library: LibraryStrings{
		SectionLabel: "Библиотека",
		AddURLTitle:  "Сохранить URL в библиотеку",
		AddURLPrompt: "URL:",
		AddURLHint:   "enter сохранить · esc отмена",
		Saved:        "сохранено в библиотеку",
		Fetching:     "загрузка сохранённого URL…",
		FetchFailed:  "не удалось загрузить URL",
		InvalidURL:   "неверный URL",
		Deleted:      "удалено из библиотеки",
	},
}

// For returns a pointer to the string table for the given language.
// Unknown languages fall back to English.
func For(l Lang) *Strings {
	switch l {
	case RU:
		return &ru
	default:
		return &en
	}
}

// Parse normalizes a stored string (e.g. from the DB settings table) into
// a known Lang. Anything other than "ru" is treated as English.
func Parse(s string) Lang {
	if s == "ru" {
		return RU
	}
	return EN
}
