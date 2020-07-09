package sqlite3perf

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// nolint:gochecknoglobals
var (
	cfgFile string
	dbPath  string

	// rootCmd represents the base command when called without any subcommands
	rootCmd = &cobra.Command{
		Use:   "sqlite3perf",
		Short: "small application to judge Golang's performance with SQLite3",
		Long: `This application was built while researching the answer to
the question https://stackoverflow.com/questions/48000940/.

It consists of two parts: the go binary you just called and 'bench.py',
which is as much of a Python implementation of the 'bench' command as I am
capable of (improvements more than welcome!).

You first have to fill the database with the 'generate' command, after which you
can call the 'bench' command to see how Go performs with SQLite3.

After the database is filled, one can run 'bench.py' against it
	`,
		// Uncomment the following line if your bare application
		// has an action associated with it:
		//	Run: func(cmd *cobra.Command, args []string) { },
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// nolint:gochecknoinits,wsl
func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&cfgFile, "config", "", "config file (default is $HOME/.sqlite3perf.yaml)")
	pf.StringVarP(&dbPath, "db", "d", "./sqlite3perf.db", "path to database")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".sqlite3perf" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".sqlite3perf")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
