package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	version = "0.0.3"
	rootCmd = &cobra.Command{
		Use:     "",
		Short:   "Go Download Manager",
		Version: version,
		Args:    cobra.MaximumNArgs(1),
		Run:     DownloadFunc,
	}
	concurrencyFlag     int  // 并发数
	resumeFlag          bool // 断点续传
	urlFlag, outputFlag string
)

func init() {
	concurrencyFlag = runtime.NumCPU()

	rootCmd.PersistentFlags().StringVarP(&urlFlag, "url", "u", "", "-u url")
	rootCmd.PersistentFlags().StringVarP(&outputFlag, "output", "o", "", "-o ")
	rootCmd.PersistentFlags().IntVarP(&concurrencyFlag, "concurrency", "n", concurrencyFlag, "-n num")
	rootCmd.PersistentFlags().BoolVarP(&resumeFlag, "resume", "r", true, "")

	// viper.BindPFlag("url", rootCmd.PersistentFlags().Lookup("url"))
	// viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	// viper.BindPFlag("concurrency", rootCmd.PersistentFlags().Lookup("concurrency"))
	// viper.BindPFlag("resume", rootCmd.PersistentFlags().Lookup("resume"))

	rootCmd.AddCommand(versionCmd)
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func CheckErr(msg interface{}) {
	if msg != nil {
		fmt.Fprintln(os.Stderr, "Error:", msg)
		os.Exit(1)
	}
}
