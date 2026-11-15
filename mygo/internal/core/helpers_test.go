package core

import "github.com/ixxxxoooo/minicode/internal/diag"

func countOf(name string) int64 { return diag.NewCounter(name).Load() }
