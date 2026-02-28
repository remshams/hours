package theme

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestScalarColorFieldsCount verifies that scalarColorFields returns exactly
// the 28 scalar (non-slice) color fields declared on Theme.
func TestScalarColorFieldsCount(t *testing.T) {
	fields := scalarColorFields(Theme{})
	assert.Len(t, fields, 28)
}

// TestScalarColorFieldsNames verifies that every expected JSON field name is
// present in the slice returned by scalarColorFields.
func TestScalarColorFieldsNames(t *testing.T) {
	fields := scalarColorFields(Theme{})

	names := make([]string, 0, len(fields))
	for _, f := range fields {
		names = append(names, f.name)
	}

	expected := []string{
		"activeTask",
		"activeTaskBeginTime",
		"activeTasks",
		"formContext",
		"formFieldName",
		"formHelp",
		"helpMsg",
		"helpPrimary",
		"helpSecondary",
		"inactiveTasks",
		"initialHelpMsg",
		"listItemDesc",
		"listItemTitle",
		"recordsBorder",
		"recordsDateRange",
		"recordsFooter",
		"recordsHeader",
		"recordsHelp",
		"taskEntry",
		"taskLogDetails",
		"taskLogEntry",
		"taskLogFormError",
		"taskLogFormInfo",
		"taskLogFormWarn",
		"taskLogList",
		"titleForeground",
		"toolName",
		"tracking",
	}

	assert.Equal(t, expected, names)
}

// TestScalarColorFieldsValuesReflectTheme verifies that the value of each
// returned field corresponds to the matching struct field on the Theme.
func TestScalarColorFieldsValuesReflectTheme(t *testing.T) {
	t.Run("empty theme produces all empty values", func(t *testing.T) {
		fields := scalarColorFields(Theme{})
		for _, f := range fields {
			assert.Empty(t, f.value, "field %q should be empty for zero-value Theme", f.name)
		}
	})

	t.Run("populated theme values are reflected", func(t *testing.T) {
		thm := Theme{
			ActiveTask:      "#ff0000",
			TitleForeground: "#00ff00",
			Tracking:        "#0000ff",
		}
		fields := scalarColorFields(thm)

		byName := make(map[string]string, len(fields))
		for _, f := range fields {
			byName[f.name] = f.value
		}

		assert.Equal(t, "#ff0000", byName["activeTask"])
		assert.Equal(t, "#00ff00", byName["titleForeground"])
		assert.Equal(t, "#0000ff", byName["tracking"])
	})
}

// TestGetInvalidColorsScalarFields exercises the scalar-field validation path
// of getInvalidColors directly, without going through file loading.
func TestGetInvalidColorsScalarFields(t *testing.T) {
	testCases := []struct {
		name            string
		theme           Theme
		expectedInvalid []string
	}{
		{
			name: "all scalar fields valid hex colors",
			theme: Theme{
				ActiveTask:              "#8ec07c",
				ActiveTaskBeginTime:     "#d3869b",
				ActiveTasks:             "#fe8019",
				FormContext:             "#fabd2f",
				FormFieldName:           "#8ec07c",
				FormHelp:                "#928374",
				HelpMsg:                 "#83a598",
				HelpPrimary:             "#bdae93",
				HelpSecondary:           "#bdae93",
				InactiveTasks:           "#928374",
				InitialHelpMsg:          "#a58390",
				ListItemDesc:            "#777777",
				ListItemTitle:           "#dddddd",
				RecordsBorder:           "#665c54",
				RecordsDateRange:        "#fabd2f",
				RecordsFooter:           "#ef8f62",
				RecordsHeader:           "#d85d5d",
				RecordsHelp:             "#928374",
				TaskLogDetailsViewTitle: "#d3869b",
				TaskEntry:               "#8ec07c",
				TaskLogEntry:            "#fabd2f",
				TaskLogList:             "#b8bb26",
				TaskLogFormInfo:         "#d3869b",
				TaskLogFormWarn:         "#fe8019",
				TaskLogFormError:        "#fb4934",
				TitleForeground:         "#282828",
				ToolName:                "#fe8019",
				Tracking:                "#fabd2f",
			},
			expectedInvalid: nil,
		},
		{
			name: "all scalar fields valid terminal color indices",
			theme: Theme{
				ActiveTask:              "0",
				ActiveTaskBeginTime:     "1",
				ActiveTasks:             "255",
				FormContext:             "100",
				FormFieldName:           "200",
				FormHelp:                "50",
				HelpMsg:                 "128",
				HelpPrimary:             "64",
				HelpSecondary:           "32",
				InactiveTasks:           "16",
				InitialHelpMsg:          "8",
				ListItemDesc:            "4",
				ListItemTitle:           "2",
				RecordsBorder:           "254",
				RecordsDateRange:        "253",
				RecordsFooter:           "252",
				RecordsHeader:           "251",
				RecordsHelp:             "250",
				TaskLogDetailsViewTitle: "249",
				TaskEntry:               "248",
				TaskLogEntry:            "247",
				TaskLogList:             "246",
				TaskLogFormInfo:         "245",
				TaskLogFormWarn:         "244",
				TaskLogFormError:        "243",
				TitleForeground:         "242",
				ToolName:                "241",
				Tracking:                "240",
			},
			expectedInvalid: nil,
		},
		{
			name: "one invalid scalar field",
			theme: Theme{
				ActiveTask:              "not-a-color",
				ActiveTaskBeginTime:     "#d3869b",
				ActiveTasks:             "#fe8019",
				FormContext:             "#fabd2f",
				FormFieldName:           "#8ec07c",
				FormHelp:                "#928374",
				HelpMsg:                 "#83a598",
				HelpPrimary:             "#bdae93",
				HelpSecondary:           "#bdae93",
				InactiveTasks:           "#928374",
				InitialHelpMsg:          "#a58390",
				ListItemDesc:            "#777777",
				ListItemTitle:           "#dddddd",
				RecordsBorder:           "#665c54",
				RecordsDateRange:        "#fabd2f",
				RecordsFooter:           "#ef8f62",
				RecordsHeader:           "#d85d5d",
				RecordsHelp:             "#928374",
				TaskLogDetailsViewTitle: "#d3869b",
				TaskEntry:               "#8ec07c",
				TaskLogEntry:            "#fabd2f",
				TaskLogList:             "#b8bb26",
				TaskLogFormInfo:         "#d3869b",
				TaskLogFormWarn:         "#fe8019",
				TaskLogFormError:        "#fb4934",
				TitleForeground:         "#282828",
				ToolName:                "#fe8019",
				Tracking:                "#fabd2f",
			},
			expectedInvalid: []string{"activeTask"},
		},
		{
			name:  "empty theme — all scalar fields are empty strings, all invalid",
			theme: Theme{},
			expectedInvalid: func() []string {
				names := make([]string, 0, 28)
				for _, f := range scalarColorFields(Theme{}) {
					names = append(names, f.name)
				}
				return names
			}(),
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := getInvalidColors(tt.theme)
			// Only check scalar portion when there are no tasks.
			if tt.expectedInvalid == nil {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.expectedInvalid, got)
			}
		})
	}
}

// TestGetInvalidColorsTaskFields exercises the tasks-slice validation path.
func TestGetInvalidColorsTaskFields(t *testing.T) {
	validScalarTheme := Theme{
		ActiveTask:              "#8ec07c",
		ActiveTaskBeginTime:     "#d3869b",
		ActiveTasks:             "#fe8019",
		FormContext:             "#fabd2f",
		FormFieldName:           "#8ec07c",
		FormHelp:                "#928374",
		HelpMsg:                 "#83a598",
		HelpPrimary:             "#bdae93",
		HelpSecondary:           "#bdae93",
		InactiveTasks:           "#928374",
		InitialHelpMsg:          "#a58390",
		ListItemDesc:            "#777777",
		ListItemTitle:           "#dddddd",
		RecordsBorder:           "#665c54",
		RecordsDateRange:        "#fabd2f",
		RecordsFooter:           "#ef8f62",
		RecordsHeader:           "#d85d5d",
		RecordsHelp:             "#928374",
		TaskLogDetailsViewTitle: "#d3869b",
		TaskEntry:               "#8ec07c",
		TaskLogEntry:            "#fabd2f",
		TaskLogList:             "#b8bb26",
		TaskLogFormInfo:         "#d3869b",
		TaskLogFormWarn:         "#fe8019",
		TaskLogFormError:        "#fb4934",
		TitleForeground:         "#282828",
		ToolName:                "#fe8019",
		Tracking:                "#fabd2f",
	}

	testCases := []struct {
		name            string
		tasks           []string
		expectedInvalid []string
	}{
		{
			name:            "no tasks",
			tasks:           nil,
			expectedInvalid: nil,
		},
		{
			name:            "all valid task colors",
			tasks:           []string{"#d3869b", "#b8bb26", "0", "255"},
			expectedInvalid: nil,
		},
		{
			name:            "first task invalid",
			tasks:           []string{"not-a-color", "#b8bb26"},
			expectedInvalid: []string{"tasks[1]"},
		},
		{
			name:            "second task invalid",
			tasks:           []string{"#d3869b", "not-a-color"},
			expectedInvalid: []string{"tasks[2]"},
		},
		{
			name:            "multiple tasks invalid",
			tasks:           []string{"#d3869b", "bad", "#90e0ef", "also-bad"},
			expectedInvalid: []string{"tasks[2]", "tasks[4]"},
		},
		{
			name:            "out-of-range terminal index is invalid",
			tasks:           []string{"256"},
			expectedInvalid: []string{"tasks[1]"},
		},
		{
			name:            "negative terminal index is invalid",
			tasks:           []string{"-1"},
			expectedInvalid: []string{"tasks[1]"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			thm := validScalarTheme
			thm.Tasks = tt.tasks

			got := getInvalidColors(thm)
			if tt.expectedInvalid == nil {
				assert.Empty(t, got)
			} else {
				assert.Equal(t, tt.expectedInvalid, got)
			}
		})
	}
}
