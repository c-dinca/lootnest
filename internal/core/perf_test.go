package core

import (
	"math"
	"testing"
)

func TestGPUScore(t *testing.T) {
	for name, want := range map[string]int{
		"NVIDIA GeForce RTX 3070":            13500,
		"NVIDIA GEFORCE GTX 1060 3 GB":       3900,
		"GeForce GTX 1060 6GB":               4200,
		"GTX970":                             3400,
		"Nvidia RTX 3060Ti":                  11500,
		"AMD Radeon RX 6700 XT":              12500,
		"AMD RADEON RX 580 4 GB":             4400,
		"Radeon RX Vega 56 8 GB":             6000,
		"AMD Radeon RX5700XT":                9200,
		"NVIDIA GeForce RTX 4060 Laptop GPU": 7875,
		"Intel(R) UHD Graphics 630":          700,
		"Intel(R) Iris(R) Xe Graphics":       1600,
		"AMD Radeon(TM) Graphics":            1500,
		"AMD Radeon 780M":                    2900,
		"Intel(R) Arc(TM) A770 Graphics":     13500,
		"Microsoft Basic Display Adapter":    0,
		"A graphics card with DirectX 11":    0,
	} {
		if got := GPUScore(name); got != want {
			t.Errorf("GPUScore(%q) = %d, want %d (keys %q)", name, got, want, gpuKeys(name))
		}
	}
}

func TestCPUScore(t *testing.T) {
	near := func(a, b float64) bool { return math.Abs(a-b) < 0.01 }
	for name, want := range map[string]float64{
		"INTEL CORE I5-8400":                        9.2,
		"AMD Ryzen 7 5800X 8-Core Processor":        28 * 1.04,
		"AMD RYZEN 3 3300X":                         12 * 1.04,
		"Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz": 18.5 * 0.85,
		"Intel Core i5-2500K":                       4.1,
		"12th Gen Intel(R) Core(TM) i7-12700K":      34,
		"Intel Core i7-920":                         3.5,
		"AMD FX-8350":                               6,
		"AMD Ryzen 7 7800X3D 8-Core Processor":      34.5,
		"Intel(R) Core(TM) Ultra 7 265K":            58,
		"AMD Ryzen 5 5500U with Radeon Graphics":    21.9 * 0.6,
		"Dual core 2.4 GHz":                         0,
	} {
		if got := CPUScore(name); !near(got, want) {
			t.Errorf("CPUScore(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestWillItRun(t *testing.T) {
	min := []ReqLine{{Label: "Processor", Value: "INTEL CORE I5-8400 or AMD RYZEN 3 3300X"}, {Label: "Memory", Value: "12 GB RAM"},
		{Label: "Graphics", Value: "NVIDIA GEFORCE GTX 1060 3 GB or AMD RADEON RX 580 4 GB"}, {Label: "Storage", Value: "60 GB available space"}}
	rec := []ReqLine{{Label: "Processor", Value: "INTEL CORE I7-8700K or AMD RYZEN 5 3600X"}, {Label: "Memory", Value: "16 GB RAM"},
		{Label: "Graphics", Value: "NVIDIA GEFORCE GTX 1070 8 GB or AMD RADEON RX VEGA 56 8 GB"}, {Label: "Storage", Value: "60 GB available space"}}
	pc := PCSpecs{CPU: "AMD Ryzen 7 5800X 8-Core Processor", GPU: "NVIDIA GeForce RTX 3070", RAM: 16 << 30, Free: 412 << 30}

	v := WillItRun(pc, min, rec)
	if v.Runs != RunsHigh || v.Limit != "" {
		t.Fatalf("a strong PC: %+v", v)
	}
	if g := v.Parts[0]; g.Min != 3900 || g.Rec != 5900 || g.Runs != RunsHigh {
		t.Fatalf("graphics: %+v (the weaker of the two cards counts)", g)
	}

	pc.GPU = "NVIDIA GeForce GTX 1060 6GB"
	if v := WillItRun(pc, min, rec); v.Runs != RunsMinimum || v.Limit != "Graphics" {
		t.Fatalf("a GTX 1060: %+v", v)
	}
	pc.GPU = "NVIDIA GeForce GTX 1050 Ti"
	if v := WillItRun(pc, min, rec); v.Runs != RunsBelow || v.Limit != "Graphics" {
		t.Fatalf("a GTX 1050 Ti: %+v", v)
	}
	pc.GPU, pc.RAM = "NVIDIA GeForce RTX 3070", 8<<30
	if v := WillItRun(pc, min, rec); v.Runs != RunsBelow || v.Limit != "Memory" {
		t.Fatalf("8 GB of memory: %+v", v)
	}
	pc.RAM, pc.Free = 16<<30, 20<<30
	if v := WillItRun(pc, min, rec); v.Runs != RunsBelow || v.Limit != "Storage" {
		t.Fatalf("not enough room: %+v", v)
	}
	// Names Lootnest doesn't know: no verdict from memory and space alone.
	pc.Free = 400 << 30
	if v := WillItRun(PCSpecs{GPU: "Mystery GPU", CPU: "Mystery CPU", RAM: 32 << 30, Free: 400 << 30}, min, rec); v.Runs != RunsUnknown || v.Parts[0].Known || !v.Parts[2].Known {
		t.Fatalf("unknown parts, enough memory and space: %+v", v)
	}
	if v := WillItRun(pc, nil, nil); v.Runs != RunsUnknown {
		t.Fatalf("no requirements: %+v", v)
	}
}

func TestPlanRun(t *testing.T) {
	part := func(name string, have, min, rec float64, runs string) PartVerdict {
		return PartVerdict{Part: name, Have: have, Min: min, Rec: rec, Runs: runs, Known: runs != RunsUnknown}
	}
	cases := []struct {
		name   string
		v      RunVerdict
		score  int
		preset string
	}{
		{"nothing known", RunVerdict{}, 0, ""},
		{"at the recommended", RunVerdict{Parts: []PartVerdict{part("Graphics", 100, 50, 100, RunsRecommended), part("Processor", 100, 50, 100, RunsRecommended)}}, 70, "High"},
		{"well above", RunVerdict{Parts: []PartVerdict{part("Graphics", 200, 50, 100, RunsHigh), part("Processor", 150, 50, 100, RunsHigh)}}, 100, "Ultra"},
		{"the weaker part decides", RunVerdict{Parts: []PartVerdict{part("Graphics", 200, 50, 100, RunsHigh), part("Processor", 60, 50, 100, RunsMinimum)}}, 42, "Low"},
		{"only a minimum", RunVerdict{Parts: []PartVerdict{part("Graphics", 50, 50, 0, RunsMinimum)}}, 40, "Low"},
		{"memory short caps it", RunVerdict{Parts: []PartVerdict{part("Graphics", 200, 50, 100, RunsHigh), part("Memory", 8, 8, 16, RunsMinimum)}}, 55, "Medium"},
		{"well under", RunVerdict{Parts: []PartVerdict{part("Graphics", 10, 50, 100, RunsBelow)}}, 7, ""},
	}
	for _, c := range cases {
		p := PlanRun(c.v)
		if p.Score != c.score || p.Preset != c.preset {
			t.Errorf("%s: %+v, want score %d preset %q", c.name, p, c.score, c.preset)
		}
	}
}
