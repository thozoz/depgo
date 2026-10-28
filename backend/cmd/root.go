package cmd

import (
	"fmt"
	"os"

	"github.com/alist-org/alist/v3/cmd/flags"
	_ "github.com/alist-org/alist/v3/drivers"
	_ "github.com/alist-org/alist/v3/internal/archive"
	_ "github.com/alist-org/alist/v3/internal/offline_download"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "depgo",
	Short: "A secure Go file server with modern web interface.",
	Long: `Depgo - Go file server and clean web workspace for local storage.
Linux-first, terminal-first deployment and operation.`,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVar(&flags.DataDir, "data", "data", "data folder")
	RootCmd.PersistentFlags().StringVar(&flags.RootFolder, "root", "", "served local root folder (or DEPGO_LOCAL_ROOT env)")
	RootCmd.PersistentFlags().BoolVar(&flags.Debug, "debug", false, "start with debug mode")
	RootCmd.PersistentFlags().BoolVar(&flags.NoPrefix, "no-prefix", false, "disable env prefix")
	RootCmd.PersistentFlags().BoolVar(&flags.Dev, "dev", false, "start with dev mode")
	RootCmd.PersistentFlags().BoolVar(&flags.ForceBinDir, "force-bin-dir", false, "Force to use the directory where the binary file is located as data directory")
	RootCmd.PersistentFlags().BoolVar(&flags.LogStd, "log-std", false, "Force to log to std")
}
