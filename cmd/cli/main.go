package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/manolis/budgeting/internal/auth"
	"github.com/manolis/budgeting/internal/config"
	"github.com/manolis/budgeting/internal/database"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	command := os.Args[1]

	switch command {
	case "user:add":
		handleUserAdd(db)
	case "user:edit":
		handleUserEdit(db)
	case "user:delete":
		handleUserDelete(db)
	case "user:list":
		handleUserList(db)
	case "actions:query":
		handleActionsQuery(db)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: cli <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  user:add       -username <username> -name <name>")
	fmt.Println("  user:edit      -username <username> [-name <name>]")
	fmt.Println("  user:delete    -username <username>")
	fmt.Println("  user:list")
	fmt.Println("  actions:query  -username <username> [-type income|expense] [-date-range YYYYMMDD-YYYYMMDD]")
}

func handleUserAdd(db *database.DB) {
	fs := flag.NewFlagSet("user:add", flag.ExitOnError)
	username := fs.String("username", "", "Username (required)")
	name := fs.String("name", "", "Display name (required)")
	fs.Parse(os.Args[2:])

	if *username == "" || *name == "" {
		fmt.Println("Error: -username and -name are required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	password := readPassword("Enter password (leave empty to generate): ")
	var hashedPassword string
	var plainPassword string

	if password == "" {
		var err error
		plainPassword, err = auth.GenerateRandomPassword(16)
		if err != nil {
			log.Fatalf("Failed to generate password: %v", err)
		}
		hashedPassword, err = auth.HashPassword(plainPassword)
		if err != nil {
			log.Fatalf("Failed to hash password: %v", err)
		}
		fmt.Printf("Generated password: %s\n", plainPassword)
		fmt.Println("Please save this password securely. It will not be shown again.")
	} else {
		if len(password) < 16 {
			log.Fatal("Password must be at least 16 characters")
		}
		var err error
		hashedPassword, err = auth.HashPassword(password)
		if err != nil {
			log.Fatalf("Failed to hash password: %v", err)
		}
	}

	user, err := db.CreateUser(*username, hashedPassword, *name)
	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}

	fmt.Printf("User created successfully: ID=%d, Username=%s, Name=%s\n", user.ID, user.Username, user.Name)
}

func handleUserEdit(db *database.DB) {
	fs := flag.NewFlagSet("user:edit", flag.ExitOnError)
	username := fs.String("username", "", "Username (required)")
	name := fs.String("name", "", "New display name")
	fs.Parse(os.Args[2:])

	if *username == "" {
		fmt.Println("Error: -username is required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	password := readPassword("Enter new password (leave empty to keep current): ")

	var hashedPassword *string
	var newName *string

	if password != "" {
		if len(password) < 16 {
			log.Fatal("Password must be at least 16 characters")
		}
		hashed, err := auth.HashPassword(password)
		if err != nil {
			log.Fatalf("Failed to hash password: %v", err)
		}
		hashedPassword = &hashed
	}

	if *name != "" {
		newName = name
	}

	if hashedPassword == nil && newName == nil {
		fmt.Println("No changes specified")
		os.Exit(0)
	}

	if err := db.UpdateUser(*username, hashedPassword, newName); err != nil {
		log.Fatalf("Failed to update user: %v", err)
	}

	fmt.Println("User updated successfully")
}

func handleUserDelete(db *database.DB) {
	fs := flag.NewFlagSet("user:delete", flag.ExitOnError)
	username := fs.String("username", "", "Username (required)")
	fs.Parse(os.Args[2:])

	if *username == "" {
		fmt.Println("Error: -username is required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	fmt.Printf("Are you sure you want to delete user '%s'? (yes/no): ", *username)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "yes" {
		fmt.Println("Deletion cancelled")
		os.Exit(0)
	}

	if err := db.DeleteUser(*username); err != nil {
		log.Fatalf("Failed to delete user: %v", err)
	}

	fmt.Println("User deleted successfully")
}

func handleUserList(db *database.DB) {
	users, err := db.ListUsers()
	if err != nil {
		log.Fatalf("Failed to list users: %v", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tUSERNAME\tNAME\tCREATED AT\tUPDATED AT")
	fmt.Fprintln(w, "---\t--------\t----\t----------\t----------")

	for _, user := range users {
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			user.ID,
			user.Username,
			user.Name,
			user.CreatedAt.Format("2006-01-02 15:04:05"),
			user.UpdatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	w.Flush()
}

func handleActionsQuery(db *database.DB) {
	fs := flag.NewFlagSet("actions:query", flag.ExitOnError)
	username := fs.String("username", "", "Username (required)")
	actionType := fs.String("type", "", "Action type (income|expense)")
	dateRange := fs.String("date-range", "", "Date range (YYYYMMDD-YYYYMMDD)")
	fs.Parse(os.Args[2:])

	if *username == "" {
		fmt.Println("Error: -username is required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	filters := database.ActionFilters{
		Username: *username,
		Type:     *actionType,
	}

	if *dateRange != "" {
		parts := strings.Split(*dateRange, "-")
		if len(parts) != 2 {
			log.Fatal("Invalid date range format. Use YYYYMMDD-YYYYMMDD")
		}
		// Convert YYYYMMDD to YYYY-MM-DD
		if len(parts[0]) == 8 {
			filters.DateFrom = parts[0][:4] + "-" + parts[0][4:6] + "-" + parts[0][6:8]
		}
		if len(parts[1]) == 8 {
			filters.DateTo = parts[1][:4] + "-" + parts[1][4:6] + "-" + parts[1][6:8]
		}
	}

	actions, err := db.ListActions(filters)
	if err != nil {
		log.Fatalf("Failed to query actions: %v", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DATE\tTYPE\tDESCRIPTION\tAMOUNT\tCREATED AT")
	fmt.Fprintln(w, "----\t----\t-----------\t------\t----------")

	for _, action := range actions {
		fmt.Fprintf(w, "%s\t%s\t%s\t%.2f\t%s\n",
			action.Date,
			action.Type,
			action.Description,
			action.Amount,
			action.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}

	w.Flush()
}

func readPassword(prompt string) string {
	fmt.Print(prompt)
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(bytePassword))
}
