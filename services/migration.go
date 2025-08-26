package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MigrateChannelsToMap migrates channels from array to map structure in both vector_documents and companies collections
func MigrateChannelsToMap(ctx context.Context) error {
	db := GetDatabase()

	// Migrate vector_documents collection
	if err := migrateVectorDocuments(ctx, db); err != nil {
		return fmt.Errorf("failed to migrate vector_documents: %w", err)
	}

	// Migrate companies collection (CRM links)
	if err := migrateCompanies(ctx, db); err != nil {
		return fmt.Errorf("failed to migrate companies: %w", err)
	}

	return nil
}

// migrateVectorDocuments migrates the channels field in vector_documents from array to map
func migrateVectorDocuments(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("vector_documents")

	// Find all documents with channels as array (old format)
	filter := bson.M{
		"channels": bson.M{
			"$type": "array",
		},
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to find documents: %w", err)
	}
	defer cursor.Close(ctx)

	var migratedCount int64
	var failedCount int64

	for cursor.Next(ctx) {
		var doc struct {
			ID       interface{} `bson:"_id"`
			Channels []string    `bson:"channels"`
		}

		if err := cursor.Decode(&doc); err != nil {
			slog.Error("Failed to decode document", "error", err)
			failedCount++
			continue
		}

		// Convert array to map
		channelMap := make(map[string]bool)
		// Initialize both channels as false
		channelMap["facebook"] = false
		channelMap["messenger"] = false

		// Set channels that were in the array to true
		for _, ch := range doc.Channels {
			ch = normalizeChannel(ch)
			if ch == "facebook" || ch == "messenger" {
				channelMap[ch] = true
			}
		}

		// Update the document
		updateFilter := bson.M{"_id": doc.ID}
		update := bson.M{
			"$set": bson.M{
				"channels":   channelMap,
				"updated_at": time.Now(),
			},
		}

		if _, err := collection.UpdateOne(ctx, updateFilter, update); err != nil {
			slog.Error("Failed to update document",
				"id", doc.ID,
				"error", err)
			failedCount++
			continue
		}

		migratedCount++
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error: %w", err)
	}

	slog.Info("Vector documents migration completed",
		"migrated", migratedCount,
		"failed", failedCount)

	return nil
}

// migrateCompanies migrates the channels field in CRM links from array to map
func migrateCompanies(ctx context.Context, db *mongo.Database) error {
	collection := db.Collection("companies")

	// Find all companies
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return fmt.Errorf("failed to find companies: %w", err)
	}
	defer cursor.Close(ctx)

	var migratedCount int64
	var failedCount int64

	for cursor.Next(ctx) {
		var company struct {
			ID    interface{} `bson:"_id"`
			Pages []struct {
				PageID   string `bson:"page_id"`
				CRMLinks []struct {
					Channels interface{} `bson:"channels"`
				} `bson:"crm_links,omitempty"`
				FacebookConfig *struct {
					CRMLinks []struct {
						Channels interface{} `bson:"channels"`
					} `bson:"crm_links,omitempty"`
				} `bson:"facebook_config,omitempty"`
				MessengerConfig *struct {
					CRMLinks []struct {
						Channels interface{} `bson:"channels"`
					} `bson:"crm_links,omitempty"`
				} `bson:"messenger_config,omitempty"`
			} `bson:"pages"`
		}

		if err := cursor.Decode(&company); err != nil {
			slog.Error("Failed to decode company", "error", err)
			failedCount++
			continue
		}

		updates := bson.M{
			"updated_at": time.Now(),
		}
		needsUpdate := false

		// Process each page
		for pageIdx, page := range company.Pages {
			// Migrate legacy CRMLinks
			for linkIdx, link := range page.CRMLinks {
				if channelsArray, ok := link.Channels.([]interface{}); ok {
					channelMap := convertArrayToMap(channelsArray)
					updates[fmt.Sprintf("pages.%d.crm_links.%d.channels", pageIdx, linkIdx)] = channelMap
					needsUpdate = true
				}
			}

			// Migrate FacebookConfig CRMLinks
			if page.FacebookConfig != nil {
				for linkIdx, link := range page.FacebookConfig.CRMLinks {
					if channelsArray, ok := link.Channels.([]interface{}); ok {
						channelMap := convertArrayToMap(channelsArray)
						updates[fmt.Sprintf("pages.%d.facebook_config.crm_links.%d.channels", pageIdx, linkIdx)] = channelMap
						needsUpdate = true
					}
				}
			}

			// Migrate MessengerConfig CRMLinks
			if page.MessengerConfig != nil {
				for linkIdx, link := range page.MessengerConfig.CRMLinks {
					if channelsArray, ok := link.Channels.([]interface{}); ok {
						channelMap := convertArrayToMap(channelsArray)
						updates[fmt.Sprintf("pages.%d.messenger_config.crm_links.%d.channels", pageIdx, linkIdx)] = channelMap
						needsUpdate = true
					}
				}
			}
		}

		// Update if needed
		if needsUpdate {
			updateFilter := bson.M{"_id": company.ID}
			update := bson.M{"$set": updates}

			if _, err := collection.UpdateOne(ctx, updateFilter, update); err != nil {
				slog.Error("Failed to update company",
					"id", company.ID,
					"error", err)
				failedCount++
				continue
			}

			migratedCount++
		}
	}

	if err := cursor.Err(); err != nil {
		return fmt.Errorf("cursor error: %w", err)
	}

	slog.Info("Companies migration completed",
		"migrated", migratedCount,
		"failed", failedCount)

	return nil
}

// convertArrayToMap converts a channels array to a map
func convertArrayToMap(channelsArray []interface{}) map[string]bool {
	channelMap := make(map[string]bool)
	// Initialize both channels as false
	channelMap["facebook"] = false
	channelMap["messenger"] = false

	// Set channels that were in the array to true
	for _, ch := range channelsArray {
		if chStr, ok := ch.(string); ok {
			chStr = normalizeChannel(chStr)
			if chStr == "facebook" || chStr == "messenger" {
				channelMap[chStr] = true
			}
		}
	}

	return channelMap
}
