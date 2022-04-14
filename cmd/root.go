package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:     "",
		Short:   "Go Download Manager",
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			url, _ := cmd.Flags().GetString("url")
			filename, _ := cmd.Flags().GetString("output")
			concurrency, _ := cmd.Flags().GetInt("concurrency")
			resume, _ := cmd.Flags().GetBool("resume")

			fmt.Printf("[url] %v\n[filename] %v\n[concurrency] %v\n", url, filename, concurrency)

			NewDownloader(concurrency, resume).Download(url, filename)
		},
	}
	version = "0.0.1"
)

func init() {
	concurrencyN := runtime.NumCPU()

	var url, output string
	var resume bool
	rootCmd.PersistentFlags().StringVarP(&url, "url", "u", "", "-u url")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "", "-o ")
	rootCmd.PersistentFlags().IntVarP(&concurrencyN, "concurrency", "n", concurrencyN, "-n num")
	rootCmd.PersistentFlags().BoolVarP(&resume, "resume", "r", true, "")

	// viper.BindPFlag("url", rootCmd.PersistentFlags().Lookup("url"))
	// viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	// viper.BindPFlag("concurrency", rootCmd.PersistentFlags().Lookup("concurrency"))
	// viper.BindPFlag("resume", rootCmd.PersistentFlags().Lookup("resume"))

	rootCmd.AddCommand(versionCmd)
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
