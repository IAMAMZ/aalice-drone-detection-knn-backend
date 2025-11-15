package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"song-recognition/drone"
	"song-recognition/utils"
)

// Check which prototype source files actually exist
func main() {
	modelPath := utils.GetEnv("DRONE_MODEL_PATH", filepath.Join("drone", "prototypes.json"))

	data, err := os.ReadFile(modelPath)
	if err != nil {
		log.Fatalf("Failed to read prototypes: %v", err)
	}

	var prototypes []drone.Prototype
	if err := json.Unmarshal(data, &prototypes); err != nil {
		log.Fatalf("Failed to parse prototypes: %v", err)
	}

	fmt.Println("=== Checking Prototype Source Files ===")
	fmt.Printf("Total prototypes: %d\n\n", len(prototypes))

	existCount := 0
	missingCount := 0

	for i, proto := range prototypes {
		fmt.Printf("%d. %s (label: %s)\n", i+1, proto.ID, proto.Label)
		fmt.Printf("   Source: %s\n", proto.Source)

		if proto.Source == "" {
			fmt.Printf("   ⚠️  No source specified\n")
			missingCount++
		} else {
			if _, err := os.Stat(proto.Source); os.IsNotExist(err) {
				fmt.Printf("   ❌ FILE DOES NOT EXIST\n")
				missingCount++
			} else {
				fmt.Printf("   ✅ File exists\n")
				existCount++
			}
		}
		fmt.Println()
	}

	fmt.Println("=== Summary ===")
	fmt.Printf("✅ Source files exist: %d\n", existCount)
	fmt.Printf("❌ Source files missing: %d\n", missingCount)
	fmt.Println()

	if missingCount > 0 {
		fmt.Println("🔴 PROBLEM: You cannot test with the same dataset because the original files are missing!")
		fmt.Println()
		fmt.Println("This is why you're not getting near 100% confidence.")
		fmt.Println("The files in frontendrecording/ are DIFFERENT recordings.")
		fmt.Println()
		fmt.Println("To test properly:")
		fmt.Println("1. Find or recreate the original training files")
		fmt.Println("2. Put them back in train_data/")
		fmt.Println("3. Test with those exact files")
		fmt.Println("4. THEN you should get 85-95% confidence")
	} else {
		fmt.Println("✅ All source files exist!")
		fmt.Println("You CAN test with the original files to verify 100% match.")
	}
}
