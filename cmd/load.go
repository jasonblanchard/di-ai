/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/tiktoken-go/tokenizer"
)

type EntryRecord struct {
	ID   int
	Text string
}

// loadCmd represents the load command
var loadCmd = &cobra.Command{
	Use:   "load",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath, err := cmd.Flags().GetString("file")

		if err != nil {
			return fmt.Errorf("error getting file flag: %w", err)
		}

		fmt.Println(filePath)

		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("error getting opening file: %w", err)
		}

		csvReader := csv.NewReader(file)

		records, err := csvReader.ReadAll()
		if err != nil {
			return fmt.Errorf("error reading CSV: %w", err)
		}

		entryRecords := []EntryRecord{}
		tokenSum := 0

		enc, err := tokenizer.ForModel(tokenizer.TextEmbeddingAda002)
		if err != nil {
			return fmt.Errorf("error creating tokenizer: %w", err)
		}

		for _, record := range records[1:] {
			id, err := strconv.Atoi(record[0])
			text := record[1]

			if err != nil {
				return fmt.Errorf("error converting ID to int: %w", err)
			}

			_, tokens, err := enc.Encode(text)

			tokenSum = tokenSum + len(tokens)

			if err != nil {
				return fmt.Errorf("error getting tokens: %w", err)
			}

			entryRecords = append(entryRecords, EntryRecord{
				ID:   id,
				Text: text,
			})
		}

		fmt.Printf("total records: %v\n", len(entryRecords))
		fmt.Printf("total tokens: %v\n", tokenSum)

		costPer1000 := 0.0004
		cost := float64((tokenSum / 1000)) * costPer1000

		fmt.Printf("expected cost: $%v\n", cost)

		// TODO: Count tokens and estimate cost

		// For each record
		// create embedding
		// load id, embedding and full text into vector DB

		// TODO: Error handling - what happens if it fails mid-way through?
		// Check to see if we've created the embedding first before creating, probably

		return nil
	},
}

func init() {
	rootCmd.AddCommand(loadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// loadCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// loadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	loadCmd.Flags().StringP("file", "f", "", "CSV file with entries")
}
