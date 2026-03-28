package main

import (
	"fmt"
	"os"
	"time"

	"foodlist/mstodo"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "foodlist",
	Short: "Foodlist backend server and CLI",
	Long:  `Foodlist is a collaborative todo list application with real-time updates.`,
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Microsoft Todo to get a refresh token",
	Run: func(cmd *cobra.Command, args []string) {
		clientID := os.Getenv("CLIENT_ID")
		if clientID == "" {
			fmt.Println("Error: CLIENT_ID environment variable must be set")
			os.Exit(1)
		}

		port := "3000"

		if err := mstodo.Login(clientID, port); err != nil {
			fmt.Printf("Login failed: %v\n", err)
			os.Exit(1)
		}
	},
}

var mstodoCmd = &cobra.Command{
	Use:   "mstodo",
	Short: "Microsoft Todo integration",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		_ = godotenv.Load()
	},
}

var listsCmd = &cobra.Command{
	Use:   "lists",
	Short: "List all task lists",
	Run: func(cmd *cobra.Command, args []string) {
		clientID := os.Getenv("CLIENT_ID")
		refreshToken := os.Getenv("REFRESH_TOKEN")

		if clientID == "" || refreshToken == "" {
			fmt.Println("Error: CLIENT_ID and REFRESH_TOKEN environment variables must be set")
			os.Exit(1)
		}

		client := mstodo.NewClient(clientID, refreshToken)
		lists, err := client.GetLists()
		if err != nil {
			fmt.Printf("Error fetching lists: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("%-36s  %s\n", "ID", "Name")
		fmt.Println("----------------------------------------------------------------")
		for _, list := range lists {
			fmt.Printf("%-36s  %s\n", list.ID, list.DisplayName)
		}
	},
}

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "List tasks in a list (Active & Completed)",
	Run: func(cmd *cobra.Command, args []string) {
		clientID := os.Getenv("CLIENT_ID")
		refreshToken := os.Getenv("REFRESH_TOKEN")
		listID := os.Getenv("LIST_ID")

		if clientID == "" || refreshToken == "" || listID == "" {
			fmt.Println("Error: CLIENT_ID, REFRESH_TOKEN, and LIST_ID environment variables must be set")
			os.Exit(1)
		}

		client := mstodo.NewClient(clientID, refreshToken)
		tasks, err := client.GetTasks(listID)
		if err != nil {
			fmt.Printf("Error fetching tasks: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Active Todos:")
		fmt.Printf("%-12s  %s\n", "Date", "Name")
		fmt.Println("------------------------------------------------")
		for _, task := range tasks {
			if task.Status != "completed" {
				created, _ := time.Parse(time.RFC3339, task.CreatedDateTime)
				fmt.Printf("%-12s  %s\n", created.Format("2006-01-02"), task.Title)
			}
		}

		fmt.Println("\nCompleted Todos:")
		fmt.Printf("%-12s  %s\n", "Date", "Name")
		fmt.Println("------------------------------------------------")
		for _, task := range tasks {
			if task.Status == "completed" {
				dateStr := task.CreatedDateTime
				if task.CompletedDateTime != nil {
					dateStr = task.CompletedDateTime.DateTime
				}

				date, err := time.Parse(time.RFC3339, dateStr)
				if err != nil {
					if len(dateStr) >= 10 {
						dateStr = dateStr[:10]
						date, _ = time.Parse("2006-01-02", dateStr)
					}
				}

				fmt.Printf("%-12s  %s\n", date.Format("2006-01-02"), task.Title)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(mstodoCmd)
	mstodoCmd.AddCommand(listsCmd)
	mstodoCmd.AddCommand(tasksCmd)
	mstodoCmd.AddCommand(loginCmd)
}

func runCLI() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
