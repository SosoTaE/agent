package services

import (
	"context"
	"facebook-bot/models"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetRecentNameChanges retrieves recent name changes for the specified pages
func GetRecentNameChanges(ctx context.Context, pageIDs []string, limit int) ([]models.NameChange, error) {
	collection := database.Collection("name_changes")

	filter := bson.M{}
	if len(pageIDs) > 0 {
		filter["page_id"] = bson.M{"$in": pageIDs}
	}

	opts := options.Find().SetSort(bson.M{"changed_at": -1}).SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var changes []models.NameChange
	if err := cursor.All(ctx, &changes); err != nil {
		return nil, err
	}

	return changes, nil
}

// GetNameHistory retrieves the name history for a specific sender
func GetNameHistory(ctx context.Context, senderID string, pageID string) (*models.NameHistory, error) {
	collection := database.Collection("name_histories")

	filter := bson.M{
		"sender_id": senderID,
		"page_id":   pageID,
	}

	var history models.NameHistory
	err := collection.FindOne(ctx, filter).Decode(&history)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Create new history if not exists
			history = models.NameHistory{
				ID:        primitive.NewObjectID(),
				SenderID:  senderID,
				PageID:    pageID,
				Names:     []models.NameRecord{},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			return &history, nil
		}
		return nil, err
	}

	return &history, nil
}

// GetSenderNameChanges retrieves name changes for a specific sender
func GetSenderNameChanges(ctx context.Context, senderID string, pageID string) ([]models.NameChange, error) {
	collection := database.Collection("name_changes")

	filter := bson.M{
		"sender_id": senderID,
		"page_id":   pageID,
	}

	opts := options.Find().SetSort(bson.M{"changed_at": -1})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var changes []models.NameChange
	if err := cursor.All(ctx, &changes); err != nil {
		return nil, err
	}

	return changes, nil
}

// UpdateNameHistory updates or creates a name history record
func UpdateNameHistory(ctx context.Context, senderID, pageID, name string) error {
	collection := database.Collection("name_histories")

	filter := bson.M{
		"sender_id": senderID,
		"page_id":   pageID,
	}

	now := time.Now()

	var history models.NameHistory
	err := collection.FindOne(ctx, filter).Decode(&history)
	if err == mongo.ErrNoDocuments {
		// Create new history
		history = models.NameHistory{
			ID:       primitive.NewObjectID(),
			SenderID: senderID,
			PageID:   pageID,
			Names: []models.NameRecord{
				{
					Name:      name,
					FirstSeen: now,
					LastSeen:  now,
					Count:     1,
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		}
		_, err = collection.InsertOne(ctx, history)
		return err
	} else if err != nil {
		return err
	}

	// Update existing history
	nameFound := false
	for i, nameRecord := range history.Names {
		if nameRecord.Name == name {
			history.Names[i].LastSeen = now
			history.Names[i].Count++
			nameFound = true
			break
		}
	}

	if !nameFound {
		// Detect name change
		if len(history.Names) > 0 {
			lastUsedName := history.Names[len(history.Names)-1]
			if lastUsedName.Name != name {
				// Record name change
				change := models.NameChange{
					SenderID:  senderID,
					PageID:    pageID,
					OldName:   lastUsedName.Name,
					NewName:   name,
					ChangedAt: now,
				}
				SaveNameChange(ctx, change)
			}
		}

		// Add new name record
		history.Names = append(history.Names, models.NameRecord{
			Name:      name,
			FirstSeen: now,
			LastSeen:  now,
			Count:     1,
		})
	}

	history.UpdatedAt = now

	// Update the document
	replaceOpts := options.Replace().SetUpsert(true)
	_, err = collection.ReplaceOne(ctx, filter, history, replaceOpts)
	return err
}

// SaveNameChange saves a detected name change
func SaveNameChange(ctx context.Context, change models.NameChange) error {
	collection := database.Collection("name_changes")
	_, err := collection.InsertOne(ctx, change)
	return err
}
