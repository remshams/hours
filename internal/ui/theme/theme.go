package theme

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	defaultThemeName  = "default"
	CustomThemePrefix = "custom:"
)

var (
	errThemeFileIsInvalidJSON     = errors.New("theme file is not valid JSON")
	ErrThemeFileHasInvalidSchema  = errors.New("theme file's schema is incorrect")
	ErrThemeColorsAreInvalid      = errors.New("invalid colors provided")
	errCouldntReadCustomThemeFile = errors.New("couldn't read custom theme file")
	errCouldntLoadCustomTheme     = errors.New("couldn't load custom theme")
	errEmptyThemeNameProvided     = errors.New("empty theme name provided")
	ErrCustomThemeDoesntExist     = errors.New("custom theme doesn't exist")
	ErrBuiltInThemeDoesntExist    = errors.New("built-in theme doesn't exist")
)

var hexCodeRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type Theme struct {
	ActiveTask              string   `json:"activeTask,omitempty"`
	ActiveTaskBeginTime     string   `json:"activeTaskBeginTime,omitempty"`
	ActiveTasks             string   `json:"activeTasks,omitempty"`
	FormContext             string   `json:"formContext,omitempty"`
	FormFieldName           string   `json:"formFieldName,omitempty"`
	FormHelp                string   `json:"formHelp,omitempty"`
	HelpMsg                 string   `json:"helpMsg,omitempty"`
	HelpPrimary             string   `json:"helpPrimary,omitempty"`
	HelpSecondary           string   `json:"helpSecondary,omitempty"`
	InactiveTasks           string   `json:"inactiveTasks,omitempty"`
	InitialHelpMsg          string   `json:"initialHelpMsg,omitempty"`
	ListItemDesc            string   `json:"listItemDesc,omitempty"`
	ListItemTitle           string   `json:"listItemTitle,omitempty"`
	RecordsBorder           string   `json:"recordsBorder,omitempty"`
	RecordsDateRange        string   `json:"recordsDateRange,omitempty"`
	RecordsFooter           string   `json:"recordsFooter,omitempty"`
	RecordsHeader           string   `json:"recordsHeader,omitempty"`
	RecordsHelp             string   `json:"recordsHelp,omitempty"`
	TaskEntry               string   `json:"taskEntry,omitempty"`
	TaskLogDetailsViewTitle string   `json:"taskLogDetails,omitempty"`
	TaskLogEntry            string   `json:"taskLogEntry,omitempty"`
	TaskLogFormError        string   `json:"taskLogFormError,omitempty"`
	TaskLogFormInfo         string   `json:"taskLogFormInfo,omitempty"`
	TaskLogFormWarn         string   `json:"taskLogFormWarn,omitempty"`
	TaskLogList             string   `json:"taskLogList,omitempty"`
	Tasks                   []string `json:"tasks,omitempty"`
	TitleForeground         string   `json:"titleForeground,omitempty"`
	ToolName                string   `json:"toolName,omitempty"`
	Tracking                string   `json:"tracking,omitempty"`
}

func Get(themeName string, themesDir string) (Theme, error) {
	var zero Theme
	themeName = strings.TrimSpace(themeName)

	if len(themeName) == 0 {
		return zero, errEmptyThemeNameProvided
	}

	if themeName == defaultThemeName {
		return Default(), nil
	}

	if customThemeName, ok := strings.CutPrefix(themeName, CustomThemePrefix); ok {
		if len(customThemeName) == 0 {
			return zero, errEmptyThemeNameProvided
		}

		themeFilePath := filepath.Join(themesDir, fmt.Sprintf("%s.json", customThemeName))
		themeBytes, err := os.ReadFile(themeFilePath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return zero, fmt.Errorf("%w: %q", ErrCustomThemeDoesntExist, customThemeName)
			}
			return zero, fmt.Errorf("%w (%q): %s", errCouldntReadCustomThemeFile, themeFilePath, err.Error())
		}

		theme, err := loadCustom(themeBytes)
		if err != nil {
			return zero, fmt.Errorf("%w from file %q: %w", errCouldntLoadCustomTheme, themeFilePath, err)
		}

		return theme, nil
	}

	builtInTheme, err := getBuiltIn(themeName)
	if err != nil {
		return zero, err
	}

	return builtInTheme, nil
}

func Default() Theme {
	return getBuiltInTheme(paletteGruvboxDark())
}

func BuiltIn() []string {
	return []string{
		themeNameCatppuccinMocha,
		themeNameDracula,
		themeNameGithubDark,
		themeNameGruvboxDark,
		themeNameMonokaiClassic,
		themeNameNightOwl,
		themeNameTokyonight,
		themeNameXcodeDark,
	}
}

func loadCustom(themeJSON []byte) (Theme, error) {
	thm := Default()
	err := json.Unmarshal(themeJSON, &thm)
	var syntaxError *json.SyntaxError

	if err != nil {
		if errors.As(err, &syntaxError) {
			return thm, fmt.Errorf("%w: %w", errThemeFileIsInvalidJSON, err)
		}
		return thm, fmt.Errorf("%w: %s", ErrThemeFileHasInvalidSchema, err.Error())
	}

	invalidColors := getInvalidColors(thm)
	if len(invalidColors) > 0 {
		return thm, fmt.Errorf("%w: %q", ErrThemeColorsAreInvalid, invalidColors)
	}

	return thm, nil
}

func getBuiltIn(theme string) (Theme, error) {
	var palette builtInThemePalette
	switch theme {
	case themeNameCatppuccinMocha:
		palette = paletteCatppuccinMocha()
	case themeNameDracula:
		palette = paletteDracula()
	case themeNameGithubDark:
		palette = paletteGithubDark()
	case themeNameGruvboxDark:
		palette = paletteGruvboxDark()
	case themeNameMonokaiClassic:
		palette = paletteMonokaiClassic()
	case themeNameNightOwl:
		palette = paletteNightOwl()
	case themeNameTokyonight:
		palette = paletteTokyonight()
	case themeNameXcodeDark:
		palette = paletteXcodeDark()
	default:
		return Theme{}, fmt.Errorf("%w: %q", ErrBuiltInThemeDoesntExist, theme)
	}

	return getBuiltInTheme(palette), nil
}

// themeColorField associates a JSON field name with its color value for
// validation.  Using a slice of structs (rather than a map) preserves a stable
// iteration order so that error messages are always deterministic.
type themeColorField struct {
	name  string
	value string
}

// scalarColorFields returns the list of all non-slice color fields together
// with their JSON field names, in declaration order.
func scalarColorFields(t Theme) []themeColorField {
	return []themeColorField{
		{name: "activeTask", value: t.ActiveTask},
		{name: "activeTaskBeginTime", value: t.ActiveTaskBeginTime},
		{name: "activeTasks", value: t.ActiveTasks},
		{name: "formContext", value: t.FormContext},
		{name: "formFieldName", value: t.FormFieldName},
		{name: "formHelp", value: t.FormHelp},
		{name: "helpMsg", value: t.HelpMsg},
		{name: "helpPrimary", value: t.HelpPrimary},
		{name: "helpSecondary", value: t.HelpSecondary},
		{name: "inactiveTasks", value: t.InactiveTasks},
		{name: "initialHelpMsg", value: t.InitialHelpMsg},
		{name: "listItemDesc", value: t.ListItemDesc},
		{name: "listItemTitle", value: t.ListItemTitle},
		{name: "recordsBorder", value: t.RecordsBorder},
		{name: "recordsDateRange", value: t.RecordsDateRange},
		{name: "recordsFooter", value: t.RecordsFooter},
		{name: "recordsHeader", value: t.RecordsHeader},
		{name: "recordsHelp", value: t.RecordsHelp},
		{name: "taskEntry", value: t.TaskEntry},
		{name: "taskLogDetails", value: t.TaskLogDetailsViewTitle},
		{name: "taskLogEntry", value: t.TaskLogEntry},
		{name: "taskLogFormError", value: t.TaskLogFormError},
		{name: "taskLogFormInfo", value: t.TaskLogFormInfo},
		{name: "taskLogFormWarn", value: t.TaskLogFormWarn},
		{name: "taskLogList", value: t.TaskLogList},
		{name: "titleForeground", value: t.TitleForeground},
		{name: "toolName", value: t.ToolName},
		{name: "tracking", value: t.Tracking},
	}
}

func getInvalidColors(theme Theme) []string {
	var invalidColors []string

	for _, field := range scalarColorFields(theme) {
		if !isValidColor(field.value) {
			invalidColors = append(invalidColors, field.name)
		}
	}

	for i, color := range theme.Tasks {
		if !isValidColor(color) {
			invalidColors = append(invalidColors, fmt.Sprintf("tasks[%d]", i+1))
		}
	}

	return invalidColors
}

func isValidColor(s string) bool {
	if len(s) == 0 {
		return false
	}

	if strings.HasPrefix(s, "#") {
		return hexCodeRegex.MatchString(s)
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		return false
	}

	if i < 0 || i > 255 {
		return false
	}

	return true
}
