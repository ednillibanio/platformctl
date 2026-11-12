package sharedroot

import (
	"encoding/json"
	"fmt"
)

// write prints the report as text or JSON. Neither holds a secret value.
func (d *doctor) write() error {
	if d.opts.Format == "json" {
		enc := json.NewEncoder(d.opts.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(d.report); err != nil {
			return fmt.Errorf("write the report: %w", err)
		}
		return nil
	}
	for _, f := range d.report.Problems {
		fmt.Fprintf(d.opts.Out, "problem  %s\n", f)
	}
	for _, f := range d.report.Warnings {
		fmt.Fprintf(d.opts.Out, "warning  %s\n", f)
	}
	if fp := d.report.AutomationKey; fp != nil && fp.Fingerprint != "" {
		fmt.Fprintf(d.opts.Out, "automation key %s\n", fp.Fingerprint)
	}
	fmt.Fprintf(d.opts.Out, "%s: %d problem(s), %d warning(s)\n", d.envPath(), len(d.report.Problems), len(d.report.Warnings))
	return nil
}
