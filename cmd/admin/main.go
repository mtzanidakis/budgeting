package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/manolis/budgeting/internal/auth"
	"github.com/manolis/budgeting/internal/config"
	"github.com/manolis/budgeting/internal/database"
	"github.com/manolis/budgeting/internal/version"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Handle version command separately (no DB required)
	if command == "version" {
		fmt.Printf("admin version %s\n", version.Get())
		os.Exit(0)
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
	case "token:list":
		handleTokenList(db)
	case "token:add":
		handleTokenAdd(db)
	case "token:delete":
		handleTokenDelete(db)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: admin <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  version")
	fmt.Println("  user:add       -username <username> -name <name>")
	fmt.Println("  user:edit      -username <username> [-name <name>]")
	fmt.Println("  user:delete    -username <username>")
	fmt.Println("  user:list")
	fmt.Println("  actions:query  -username <username> [-type income|expense] [-date-range YYYYMMDD-YYYYMMDD]")
	fmt.Println("  token:list     -username <username>")
	fmt.Println("  token:add      -username <username> -name <label> [-expires YYYY-MM-DD]")
	fmt.Println("  token:delete   -id <token-id>")
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
		if len(password) < 6 {
			log.Fatal("Password must be at least 6 characters")
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
		if len(password) < 6 {
			log.Fatal("Password must be at least 6 characters")
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

func handleTokenList(db *database.DB) {
	fs := flag.NewFlagSet("token:list", flag.ExitOnError)
	username := fs.String("username", "", "Username (required)")
	fs.Parse(os.Args[2:])

	if *username == "" {
		fmt.Println("Error: -username is required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	user, err := db.GetUserByUsername(*username)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}

	tokens, err := db.ListAPITokensByUser(user.ID)
	if err != nil {
		log.Fatalf("Failed to list tokens: %v", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tCREATED\tLAST USED\tEXPIRES")
	fmt.Fprintln(w, "---\t----\t-------\t---------\t-------")
	for _, t := range tokens {
		lastUsed := "never"
		if t.LastUsedAt != nil {
			lastUsed = t.LastUsedAt.Format("2006-01-02 15:04:05")
		}
		expires := "never"
		if t.ExpiresAt != nil {
			expires = t.ExpiresAt.Format("2006-01-02")
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", t.ID, t.Name, t.CreatedAt.Format("2006-01-02 15:04:05"), lastUsed, expires)
	}
	w.Flush()
}

func handleTokenAdd(db *database.DB) {
	fs := flag.NewFlagSet("token:add", flag.ExitOnError)
	username := fs.String("username", "", "Username (required)")
	name := fs.String("name", "", "Token label (required)")
	expires := fs.String("expires", "", "Expiry date (YYYY-MM-DD, optional)")
	fs.Parse(os.Args[2:])

	if *username == "" || *name == "" {
		fmt.Println("Error: -username and -name are required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	user, err := db.GetUserByUsername(*username)
	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}

	var expiresAt *time.Time
	if *expires != "" {
		t, err := time.Parse("2006-01-02", *expires)
		if err != nil {
			log.Fatalf("Invalid -expires, use YYYY-MM-DD: %v", err)
		}
		t = t.Add(24*time.Hour - time.Second)
		if !t.After(time.Now()) {
			log.Fatal("Expiry must be in the future")
		}
		expiresAt = &t
	}

	raw, hash, err := auth.GenerateAPIToken()
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	tok, err := db.CreateAPIToken(user.ID, *name, hash, expiresAt)
	if err != nil {
		log.Fatalf("Failed to create token: %v", err)
	}

	fmt.Printf("Token created: ID=%d, Name=%s\n", tok.ID, tok.Name)
	fmt.Printf("Token: %s\n", raw)
	fmt.Println("Save this token securely. It will not be shown again.")
}

func handleTokenDelete(db *database.DB) {
	fs := flag.NewFlagSet("token:delete", flag.ExitOnError)
	idStr := fs.String("id", "", "Token ID (required)")
	fs.Parse(os.Args[2:])

	if *idStr == "" {
		fmt.Println("Error: -id is required")
		fs.PrintDefaults()
		os.Exit(1)
	}

	id, err := strconv.ParseInt(*idStr, 10, 64)
	if err != nil || id <= 0 {
		log.Fatal("Invalid -id")
	}

	// Look up the token to find its user_id (admin can delete any user's token).
	var userID int64
	row := db.QueryRow("SELECT user_id FROM api_tokens WHERE id = ? AND deleted_at IS NULL", id)
	if err := row.Scan(&userID); err != nil {
		log.Fatalf("Token not found: %v", err)
	}

	if err := db.SoftDeleteAPIToken(id, userID); err != nil {
		log.Fatalf("Failed to delete token: %v", err)
	}
	fmt.Println("Token deleted successfully")
}
