//nolint:testpackage // testing unexported function syncTerraformCliArgs
package runnerpool

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gruntwork-io/terragrunt/internal/cli"
	"github.com/gruntwork-io/terragrunt/internal/component"
	"github.com/gruntwork-io/terragrunt/options"
	"github.com/gruntwork-io/terragrunt/test/helpers/logger"
	"github.com/gruntwork-io/terragrunt/tf"
)

func TestSyncTerraformCliArgs(t *testing.T) {
	t.Parallel()

	workingDir := "/work/project"
	unitPath := filepath.Join(workingDir, "unit1")
	outputFolder := "/output"
	planFile := filepath.Join(outputFolder, "unit1", tf.TerraformPlanFile)
	outArg := "-out=" + planFile

	tests := []struct {
		name                string
		terraformCommand    string
		discoveryCtx        *component.DiscoveryContext
		unitCliArgs         cli.Args
		stackCliArgs        cli.Args
		outputFolder        string
		expectedArgs        cli.Args
		expectedPlanFileQty int // how many times the plan file (or -out=planfile) should appear
	}{
		{
			name:             "apply with plan file in discovery context args should not duplicate",
			terraformCommand: tf.CommandNameApply,
			discoveryCtx: &component.DiscoveryContext{
				WorkingDir: workingDir,
				Cmd:        tf.CommandNameApply,
				Args:       []string{"-input=false", planFile},
			},
			unitCliArgs:         cli.Args{tf.CommandNameApply, "-input=false", planFile},
			stackCliArgs:        cli.Args{tf.CommandNameApply, "-auto-approve", planFile},
			outputFolder:        outputFolder,
			expectedArgs:        cli.Args{tf.CommandNameApply, "-input=false", planFile, "-auto-approve"},
			expectedPlanFileQty: 1,
		},
		{
			name:             "apply without plan file in discovery context args should add it",
			terraformCommand: tf.CommandNameApply,
			discoveryCtx: &component.DiscoveryContext{
				WorkingDir: workingDir,
				Cmd:        tf.CommandNameApply,
				Args:       []string{"-input=false"},
			},
			unitCliArgs:         cli.Args{tf.CommandNameApply, "-input=false"},
			stackCliArgs:        cli.Args{tf.CommandNameApply, "-auto-approve"},
			outputFolder:        outputFolder,
			expectedArgs:        cli.Args{tf.CommandNameApply, "-input=false", "-auto-approve", planFile},
			expectedPlanFileQty: 1,
		},
		{
			name:             "apply without discovery context should add plan file",
			terraformCommand: tf.CommandNameApply,
			discoveryCtx: &component.DiscoveryContext{
				WorkingDir: workingDir,
			},
			unitCliArgs:         cli.Args{tf.CommandNameApply},
			stackCliArgs:        cli.Args{tf.CommandNameApply, "-auto-approve"},
			outputFolder:        outputFolder,
			expectedArgs:        cli.Args{tf.CommandNameApply, "-auto-approve", planFile},
			expectedPlanFileQty: 1,
		},
		{
			name:             "apply without output folder should not add plan file",
			terraformCommand: tf.CommandNameApply,
			discoveryCtx: &component.DiscoveryContext{
				WorkingDir: workingDir,
			},
			unitCliArgs:         cli.Args{tf.CommandNameApply},
			stackCliArgs:        cli.Args{tf.CommandNameApply, "-auto-approve"},
			outputFolder:        "",
			expectedArgs:        cli.Args{tf.CommandNameApply, "-auto-approve"},
			expectedPlanFileQty: 0,
		},
		{
			name:             "plan with -out arg in discovery context should not duplicate",
			terraformCommand: tf.CommandNamePlan,
			discoveryCtx: &component.DiscoveryContext{
				WorkingDir: workingDir,
				Cmd:        tf.CommandNamePlan,
				Args:       []string{"-input=false", outArg},
			},
			unitCliArgs:         cli.Args{tf.CommandNamePlan, "-input=false", outArg},
			stackCliArgs:        cli.Args{tf.CommandNamePlan, "-detailed-exitcode"},
			outputFolder:        outputFolder,
			expectedArgs:        cli.Args{tf.CommandNamePlan, "-input=false", outArg, "-detailed-exitcode"},
			expectedPlanFileQty: 1,
		},
		{
			name:             "plan without -out arg should add it",
			terraformCommand: tf.CommandNamePlan,
			discoveryCtx: &component.DiscoveryContext{
				WorkingDir: workingDir,
			},
			unitCliArgs:         cli.Args{tf.CommandNamePlan},
			stackCliArgs:        cli.Args{tf.CommandNamePlan, "-detailed-exitcode"},
			outputFolder:        outputFolder,
			expectedArgs:        cli.Args{tf.CommandNamePlan, "-detailed-exitcode", outArg},
			expectedPlanFileQty: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			unit := component.NewUnit(unitPath).WithDiscoveryContext(tt.discoveryCtx)

			unitOpts, err := options.NewTerragruntOptionsForTest(filepath.Join(unitPath, "terragrunt.hcl"))
			require.NoError(t, err)

			unitOpts.TerraformCommand = tt.terraformCommand
			unitOpts.TerraformCliArgs = tt.unitCliArgs

			unit.Execution = &component.UnitExecution{
				TerragruntOptions: unitOpts,
			}

			stackOpts, err := options.NewTerragruntOptionsForTest(filepath.Join(workingDir, "terragrunt.hcl"))
			require.NoError(t, err)

			stackOpts.OutputFolder = tt.outputFolder
			stackOpts.RootWorkingDir = workingDir
			stackOpts.TerraformCommand = tt.terraformCommand
			stackOpts.TerraformCliArgs = tt.stackCliArgs

			runner := &Runner{
				Stack: &component.Stack{
					Units: []*component.Unit{unit},
				},
			}

			l := logger.CreateLogger()
			runner.syncTerraformCliArgs(l, stackOpts)

			args := unit.Execution.TerragruntOptions.TerraformCliArgs

			// Check exact args match
			assert.Equal(t, tt.expectedArgs, args, "args mismatch")

			// Count plan file occurrences (either planFile or -out=planFile depending on command)
			searchFor := planFile
			if tt.terraformCommand == tf.CommandNamePlan {
				searchFor = outArg
			}

			count := 0

			for _, arg := range args {
				if arg == searchFor {
					count++
				}
			}

			assert.Equal(t, tt.expectedPlanFileQty, count, "plan file quantity mismatch in args: %v", args)
		})
	}
}
