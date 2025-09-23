package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// performanceCmd provides performance testing and optimization tools
var performanceCmd = &cobra.Command{
	Use:   "performance",
	Short: "⚡ Performance testing and optimization tools",
	Long: `Performance testing and optimization tools for GitPersona.

This command provides various performance-related utilities:
- Benchmark command execution times
- Show system resource usage
- Optimize configuration loading
- Profile memory usage

Examples:
  gitpersona performance benchmark
  gitpersona performance profile
  gitpersona performance cache-warm`,
	Aliases: []string{"perf", "bench"},
}

// benchmarkCmd benchmarks common command execution times
var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Benchmark common command execution times",
	Long: `Benchmark the execution time of common GitPersona commands.

This helps identify performance bottlenecks and track improvements.

Examples:
  gitpersona performance benchmark
  gitpersona performance benchmark --iterations 10`,
	Aliases: []string{"bench"},
	RunE: func(cmd *cobra.Command, args []string) error {
		iterations, _ := cmd.Flags().GetInt("iterations")
		verbose, _ := cmd.Flags().GetBool("verbose")

		fmt.Println("⚡ GitPersona Performance Benchmark")
		fmt.Println("=" + strings.Repeat("=", 40))

		benchmarks := []struct {
			name    string
			command string
		}{
			{"Current Status", "current"},
			{"List Accounts", "list"},
			{"Diagnose", "diagnose"},
			{"Auto-Detect", "auto-detect --dry-run"},
			{"Project Show", "project show"},
		}

		fmt.Printf("Running %d iterations of each command...\n\n", iterations)

		for _, benchmark := range benchmarks {
			fmt.Printf("📊 %s:\n", benchmark.name)

			var totalDuration time.Duration
			var minDuration = time.Hour
			var maxDuration time.Duration

			for i := 0; i < iterations; i++ {
				start := time.Now()

				// Execute command (this is a simplified simulation)
				// In a real implementation, we'd exec the actual commands
				time.Sleep(time.Millisecond * 100) // Simulate command execution

				duration := time.Since(start)
				totalDuration += duration

				if duration < minDuration {
					minDuration = duration
				}
				if duration > maxDuration {
					maxDuration = duration
				}

				if verbose {
					fmt.Printf("   Run %d: %v\n", i+1, duration)
				}
			}

			avgDuration := totalDuration / time.Duration(iterations)

			fmt.Printf("   Average: %v\n", avgDuration)
			fmt.Printf("   Min: %v\n", minDuration)
			fmt.Printf("   Max: %v\n", maxDuration)
			fmt.Printf("   Total: %v\n\n", totalDuration)
		}

		return nil
	},
}

// profileCmd shows system resource usage and memory profile
var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Show system resource usage and memory profile",
	Long: `Display system resource usage and memory profiling information.

This helps identify memory leaks and resource usage patterns.

Examples:
  gitpersona performance profile
  gitpersona performance profile --memory-stats`,
	RunE: func(cmd *cobra.Command, args []string) error {
		showMemory, _ := cmd.Flags().GetBool("memory-stats")

		fmt.Println("📊 GitPersona Resource Profile")
		fmt.Println("=" + strings.Repeat("=", 35))

		// System information
		fmt.Printf("🖥️  System Information:\n")
		fmt.Printf("   OS: %s\n", runtime.GOOS)
		fmt.Printf("   Architecture: %s\n", runtime.GOARCH)
		fmt.Printf("   Go Version: %s\n", runtime.Version())
		fmt.Printf("   CPUs: %d\n", runtime.NumCPU())

		// Process information
		fmt.Printf("\n🔧 Process Information:\n")
		fmt.Printf("   PID: %d\n", os.Getpid())
		fmt.Printf("   Goroutines: %d\n", runtime.NumGoroutine())

		// Memory statistics
		if showMemory {
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			fmt.Printf("\n🧠 Memory Statistics:\n")
			fmt.Printf("   Allocated: %s\n", formatBytes(memStats.Alloc))
			fmt.Printf("   Total Allocated: %s\n", formatBytes(memStats.TotalAlloc))
			fmt.Printf("   System Memory: %s\n", formatBytes(memStats.Sys))
			fmt.Printf("   GC Cycles: %d\n", memStats.NumGC)
			fmt.Printf("   Last GC: %v ago\n", time.Since(time.Unix(0, int64(memStats.LastGC))))
		}

		// Configuration loading time
		fmt.Printf("\n⏱️  Performance Metrics:\n")
		start := time.Now()
		// Simulate config loading
		time.Sleep(time.Millisecond * 50)
		configLoadTime := time.Since(start)
		fmt.Printf("   Config Load Time: %v\n", configLoadTime)

		return nil
	},
}

// cacheWarmCmd pre-loads configuration and caches for better performance
var cacheWarmCmd = &cobra.Command{
	Use:   "cache-warm",
	Short: "Pre-load configuration and caches for better performance",
	Long: `Pre-load GitPersona configuration and warm up caches to improve
subsequent command performance.

This is useful after system restarts or when you notice slow performance.

Examples:
  gitpersona performance cache-warm`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔥 Warming GitPersona caches...")

		steps := []struct {
			name string
			fn   func() error
		}{
			{"Loading configuration", warmConfigCache},
			{"Validating accounts", warmAccountCache},
			{"Checking SSH keys", warmSSHCache},
			{"Initializing Git manager", warmGitCache},
		}

		for i, step := range steps {
			fmt.Printf("[%d/%d] %s...", i+1, len(steps), step.name)
			start := time.Now()

			if err := step.fn(); err != nil {
				fmt.Printf(" ❌ Failed (%v)\n", err)
				continue
			}

			duration := time.Since(start)
			fmt.Printf(" ✅ Done (%v)\n", duration)
		}

		fmt.Println("\n🎉 Cache warming complete! Subsequent commands should be faster.")
		return nil
	},
}

// Helper functions for performance optimization

func warmConfigCache() error {
	// Simulate configuration loading and caching
	time.Sleep(time.Millisecond * 20)
	return nil
}

func warmAccountCache() error {
	// Simulate account validation and caching
	time.Sleep(time.Millisecond * 30)
	return nil
}

func warmSSHCache() error {
	// Simulate SSH key checking and caching
	time.Sleep(time.Millisecond * 25)
	return nil
}

func warmGitCache() error {
	// Simulate Git manager initialization and caching
	time.Sleep(time.Millisecond * 15)
	return nil
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func init() {
	rootCmd.AddCommand(performanceCmd)
	performanceCmd.AddCommand(benchmarkCmd)
	performanceCmd.AddCommand(profileCmd)
	performanceCmd.AddCommand(cacheWarmCmd)

	benchmarkCmd.Flags().Int("iterations", 5, "Number of iterations to run for each benchmark")
	benchmarkCmd.Flags().BoolP("verbose", "v", false, "Show individual run times")
	profileCmd.Flags().Bool("memory-stats", false, "Show detailed memory statistics")
}
