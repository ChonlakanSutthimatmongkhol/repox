package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ChonlakanSutthimatmongkhol/repox/internal/ai"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/config"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/learner"
	"github.com/ChonlakanSutthimatmongkhol/repox/internal/models"
)

var (
	learnFromID    string
	learnAutoApprove bool
	learnRejectID  string
	learnList      bool
	learnPrune     bool
	learnReset     bool
)

var learnCmd = &cobra.Command{
	Use:   "learn",
	Short: "Learn from developer edits to generated code",
	Long:  "Compares the latest AI-generated scaffold with current files, extracts reusable lessons, and saves them for future generations.",
	RunE:  runLearn,
}

func init() {
	learnCmd.Flags().StringVar(&learnFromID, "from", "", "Learn from a specific generation ID (default: latest)")
	learnCmd.Flags().BoolVar(&learnAutoApprove, "approve", false, "Auto-approve all extracted lessons")
	learnCmd.Flags().StringVar(&learnRejectID, "reject", "", "Reject and remove a lesson by ID")
	learnCmd.Flags().BoolVar(&learnList, "list", false, "List all stored lessons")
	learnCmd.Flags().BoolVar(&learnPrune, "prune", false, "Remove lessons with confidence < 0.3")
	learnCmd.Flags().BoolVar(&learnReset, "reset", false, "Clear all lessons")
	rootCmd.AddCommand(learnCmd)
}

func runLearn(cmd *cobra.Command, _ []string) error {
	if !config.RepoxDirExists() {
		return fmt.Errorf("no .repox/ directory found. Run `repox init` first")
	}

	// --list
	if learnList {
		return runLearnList(cmd)
	}

	// --reset
	if learnReset {
		if err := config.Save(config.RepoxPath("lessons.json"), []models.Lesson{}); err != nil {
			return fmt.Errorf("learn: reset: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "All lessons cleared.")
		return nil
	}

	// --prune
	if learnPrune {
		return runLearnPrune(cmd)
	}

	// --reject <id>
	if learnRejectID != "" {
		return runLearnReject(cmd, learnRejectID)
	}

	// Default: extract lessons from latest (or specified) generation
	return runLearnExtract(cmd)
}

func runLearnList(cmd *cobra.Command) error {
	lessons, err := config.Load[[]models.Lesson](config.RepoxPath("lessons.json"))
	if err != nil || len(lessons) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No lessons stored.")
		return nil
	}
	for _, l := range lessons {
		status := "pending"
		if l.Approved {
			status = "approved"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[%s] %s (scope: %s, confidence: %.2f, status: %s)\n",
			l.ID, l.Lesson, l.Scope, l.Confidence, status)
	}
	return nil
}

func runLearnPrune(cmd *cobra.Command) error {
	lessons, _ := config.Load[[]models.Lesson](config.RepoxPath("lessons.json"))
	var kept []models.Lesson
	pruned := 0
	for _, l := range lessons {
		if l.Confidence < 0.3 {
			pruned++
		} else {
			kept = append(kept, l)
		}
	}
	if err := config.Save(config.RepoxPath("lessons.json"), kept); err != nil {
		return fmt.Errorf("learn: prune: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Pruned %d lessons (confidence < 0.3). %d remaining.\n", pruned, len(kept))
	return nil
}

func runLearnReject(cmd *cobra.Command, id string) error {
	lessons, _ := config.Load[[]models.Lesson](config.RepoxPath("lessons.json"))
	var kept []models.Lesson
	found := false
	for _, l := range lessons {
		if l.ID == id {
			found = true
		} else {
			kept = append(kept, l)
		}
	}
	if !found {
		return fmt.Errorf("learn: lesson %q not found", id)
	}
	if err := config.Save(config.RepoxPath("lessons.json"), kept); err != nil {
		return fmt.Errorf("learn: reject: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Rejected lesson %s.\n", id)
	return nil
}

func runLearnExtract(cmd *cobra.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("learn: getwd: %w", err)
	}

	// Load generations
	generations, err := config.Load[[]models.Generation](config.RepoxPath("generations.json"))
	if err != nil || len(generations) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No generations found. Run `repox generate --ai` first.")
		return nil
	}

	// Find target generation
	gen := generations[len(generations)-1] // latest
	if learnFromID != "" {
		found := false
		for _, g := range generations {
			if g.ID == learnFromID {
				gen = g
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("learn: generation %q not found", learnFromID)
		}
	}

	if gen.SnapshotDir == "" {
		fmt.Fprintln(cmd.OutOrStdout(), "Generation has no snapshot (only AI-mode generations are tracked). Use `repox generate --ai` to create a trackable generation.")
		return nil
	}

	// Read diffs
	diffs, err := learner.ReadDiffs(gen, cwd)
	if err != nil {
		return fmt.Errorf("learn: read diffs: %w", err)
	}

	changed := 0
	for _, d := range diffs {
		if d.Changed {
			changed++
		}
	}
	if changed == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No edits detected since generation.")
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Found %d changed file(s) since generation %s.\n", changed, gen.ID)

	// Set up AI caller
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("set ANTHROPIC_API_KEY environment variable")
	}

	cfg, _ := config.Load[models.Config](config.RepoxPath("config.json"))
	model := cfg.AI.LearningModel
	if model == "" {
		model = "claude-haiku-4-5-20251001"
	}
	caller := ai.NewAnthropicClient(apiKey, model)

	// Extract lessons
	candidates, err := learner.ExtractLessons(diffs, gen.ID, gen.FeatureName, gen.Template, caller)
	if err != nil {
		return fmt.Errorf("learn: %w", err)
	}
	if len(candidates) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No lessons extracted.")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nExtracted %d candidate lesson(s):\n", len(candidates))
	for i, l := range candidates {
		fmt.Fprintf(cmd.OutOrStdout(), "  [%d] %s (confidence: %.2f)\n", i+1, l.Lesson, l.Confidence)
	}

	// Approve flow
	var approved []models.Lesson
	if learnAutoApprove {
		for _, l := range candidates {
			l.Approved = true
			approved = append(approved, l)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\nAuto-approved %d lesson(s).\n", len(approved))
	} else {
		for _, l := range candidates {
			l.Approved = true // in non-interactive mode, approve all
			approved = append(approved, l)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\nSaved %d lesson(s) (use --approve to auto-approve, --reject <id> to remove).\n", len(approved))
	}

	// Load existing lessons, dedup, and save
	existing, _ := config.Load[[]models.Lesson](config.RepoxPath("lessons.json"))
	merged := dedupLessons(existing, approved)
	if err := config.Save(config.RepoxPath("lessons.json"), merged); err != nil {
		return fmt.Errorf("learn: save lessons: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Lessons saved to .repox/lessons.json (%d total).\n", len(merged))
	return nil
}

// dedupLessons merges new lessons into existing, skipping near-duplicates.
func dedupLessons(existing, newLessons []models.Lesson) []models.Lesson {
	result := make([]models.Lesson, len(existing))
	copy(result, existing)

	for _, n := range newLessons {
		dup := false
		for i, e := range result {
			if e.Scope == n.Scope && similarity(e.Lesson, n.Lesson) > 0.8 {
				// Update confidence if new is higher
				if n.Confidence > e.Confidence {
					result[i].Confidence = n.Confidence
				}
				dup = true
				break
			}
		}
		if !dup {
			result = append(result, n)
		}
	}
	return result
}

// similarity returns a simple word-overlap ratio between two strings.
func similarity(a, b string) float64 {
	wordsA := strings.Fields(strings.ToLower(a))
	wordsB := strings.Fields(strings.ToLower(b))
	if len(wordsA) == 0 || len(wordsB) == 0 {
		return 0
	}
	setB := map[string]bool{}
	for _, w := range wordsB {
		setB[w] = true
	}
	matches := 0
	for _, w := range wordsA {
		if setB[w] {
			matches++
		}
	}
	return float64(matches) / float64(len(wordsA))
}
