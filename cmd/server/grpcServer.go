/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package server

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yjinjo/webb-server/pkg/webb_server"
)

// grpcServerCmd represents the grpcServer command
var grpcServerCmd = &cobra.Command{
	Use:   "grpc-server",
	Short: "Run the gRPC server",
	Long: `This command starts the gRPC server with the specified configuration.
Options:
  -a, --app-path TEXT     Go path of gRPC application [default:
                          {package}.interface.grpc:app]
  -s, --source-root PATH  Path of source root  [default: .]
  -p, --port INTEGER      Port of gRPC server  [default: 50051]
  -c, --config-file PATH  Path of config file
  -m, --module-path PATH  Additional python path
  --help                  Show this message and exit.`,
	Run: func(cmd *cobra.Command, args []string) {
		port := viper.GetInt("port")
		maxConnections := viper.GetInt("max_connections")

		server := webb_server.NewGRPCServer(port, maxConnections)

		fmt.Printf("Starting gRPC server on port %d\n", port)
		if err := server.Run(); err != nil {
			log.Fatalf("Failed to run the server: %v", err)
		}
	},
}

func init() {
	RunCmd.AddCommand(grpcServerCmd)

	grpcServerCmd.Flags().Int("port", 8080, "Port to run the server on")
	grpcServerCmd.Flags().Int("max-connections", 10, "Maximum number of connections")

	errPort := viper.BindPFlag("port", grpcServerCmd.Flags().Lookup("port"))
	if errPort != nil {
		return
	}
	errMaxConnections := viper.BindPFlag("max_connections", grpcServerCmd.Flags().Lookup("max-connections"))
	if errMaxConnections != nil {
		return
	}

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// grpcServerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// grpcServerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
