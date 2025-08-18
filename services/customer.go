package services

import (
	"context"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"facebook-bot/models"
)

// SaveOrUpdateCustomer saves or updates a customer record when they send a message
func SaveOrUpdateCustomer(ctx context.Context, customerID, customerName, firstName, lastName, pageID, pageName, companyID, lastMessage string) error {
	db := GetDatabase()
	collection := db.Collection("customers")

	now := time.Now()

	// Create filter to find existing customer for this page
	filter := bson.M{
		"customer_id": customerID,
		"page_id":     pageID,
	}

	// Prepare update document
	update := bson.M{
		"$set": bson.M{
			"customer_name": customerName,
			"first_name":    firstName,
			"last_name":     lastName,
			"page_id":       pageID,
			"page_name":     pageName,
			"company_id":    companyID,
			"last_message":  lastMessage,
			"last_seen":     now,
			"updated_at":    now,
		},
		"$inc": bson.M{
			"message_count": 1,
		},
		"$setOnInsert": bson.M{
			"customer_id": customerID,
			"first_seen":  now,
			"created_at":  now,
		},
	}

	// Use upsert to create if doesn't exist or update if exists
	opts := options.Update().SetUpsert(true)
	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		slog.Error("Failed to save/update customer",
			"customerID", customerID,
			"pageID", pageID,
			"error", err)
		return err
	}

	if result.UpsertedCount > 0 {
		slog.Info("New customer created",
			"customerID", customerID,
			"customerName", customerName,
			"pageID", pageID)
	} else {
		slog.Debug("Customer updated",
			"customerID", customerID,
			"pageID", pageID)
	}

	return nil
}

// GetCustomer retrieves a customer by their ID and page ID
func GetCustomer(ctx context.Context, customerID, pageID string) (*models.Customer, error) {
	db := GetDatabase()
	collection := db.Collection("customers")

	filter := bson.M{
		"customer_id": customerID,
		"page_id":     pageID,
	}

	var customer models.Customer
	err := collection.FindOne(ctx, filter).Decode(&customer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Customer not found
		}
		return nil, err
	}

	return &customer, nil
}

// GetCustomersByPage retrieves all customers for a specific page
func GetCustomersByPage(ctx context.Context, pageID string, limit, skip int) ([]models.Customer, int64, error) {
	db := GetDatabase()
	collection := db.Collection("customers")

	filter := bson.M{"page_id": pageID}

	// Get total count
	totalCount, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get customers with pagination
	findOptions := options.Find().
		SetSort(bson.M{"last_seen": -1}). // Sort by last seen, newest first
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var customers []models.Customer
	if err := cursor.All(ctx, &customers); err != nil {
		return nil, 0, err
	}

	return customers, totalCount, nil
}

// GetCustomersByCompany retrieves all customers for a company across all pages
func GetCustomersByCompany(ctx context.Context, companyID string, limit, skip int) ([]models.Customer, int64, error) {
	db := GetDatabase()
	collection := db.Collection("customers")

	filter := bson.M{"company_id": companyID}

	// Get total count
	totalCount, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Get customers with pagination
	findOptions := options.Find().
		SetSort(bson.M{"last_seen": -1}). // Sort by last seen, newest first
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var customers []models.Customer
	if err := cursor.All(ctx, &customers); err != nil {
		return nil, 0, err
	}

	return customers, totalCount, nil
}

// SearchCustomers searches for customers by name
func SearchCustomers(ctx context.Context, companyID, searchTerm string, limit int) ([]models.Customer, error) {
	db := GetDatabase()
	collection := db.Collection("customers")

	// Create search filter with regex for partial matching
	filter := bson.M{
		"company_id": companyID,
		"$or": []bson.M{
			{"customer_name": bson.M{"$regex": searchTerm, "$options": "i"}},
			{"first_name": bson.M{"$regex": searchTerm, "$options": "i"}},
			{"last_name": bson.M{"$regex": searchTerm, "$options": "i"}},
			{"customer_id": searchTerm}, // Exact match for customer ID
		},
	}

	findOptions := options.Find().
		SetSort(bson.M{"last_seen": -1}).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var customers []models.Customer
	if err := cursor.All(ctx, &customers); err != nil {
		return nil, err
	}

	return customers, nil
}

// GetCustomerStats retrieves statistics about customers
func GetCustomerStats(ctx context.Context, companyID string) (map[string]interface{}, error) {
	db := GetDatabase()
	collection := db.Collection("customers")

	// Total customers
	totalCustomers, err := collection.CountDocuments(ctx, bson.M{"company_id": companyID})
	if err != nil {
		return nil, err
	}

	// Active customers (last seen in 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	activeCustomers, err := collection.CountDocuments(ctx, bson.M{
		"company_id": companyID,
		"last_seen":  bson.M{"$gte": thirtyDaysAgo},
	})
	if err != nil {
		return nil, err
	}

	// New customers (created in last 7 days)
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	newCustomers, err := collection.CountDocuments(ctx, bson.M{
		"company_id": companyID,
		"created_at": bson.M{"$gte": sevenDaysAgo},
	})
	if err != nil {
		return nil, err
	}

	// Top customers by message count
	pipeline := []bson.M{
		{"$match": bson.M{"company_id": companyID}},
		{"$sort": bson.M{"message_count": -1}},
		{"$limit": 5},
		{"$project": bson.M{
			"customer_id":   1,
			"customer_name": 1,
			"page_name":     1,
			"message_count": 1,
			"last_seen":     1,
		}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var topCustomers []bson.M
	if err := cursor.All(ctx, &topCustomers); err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_customers":  totalCustomers,
		"active_customers": activeCustomers,
		"new_customers":    newCustomers,
		"top_customers":    topCustomers,
	}

	return stats, nil
}

// CreateIndexesForCustomers creates necessary indexes for the customers collection
func CreateIndexesForCustomers(ctx context.Context) error {
	db := GetDatabase()
	collection := db.Collection("customers")

	indexes := []mongo.IndexModel{
		// Compound index for unique customer per page
		{
			Keys: bson.D{
				{Key: "customer_id", Value: 1},
				{Key: "page_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		// Index for company queries
		{
			Keys: bson.D{{Key: "company_id", Value: 1}},
		},
		// Index for sorting by last seen
		{
			Keys: bson.D{{Key: "last_seen", Value: -1}},
		},
		// Text index for searching
		{
			Keys: bson.D{
				{Key: "customer_name", Value: "text"},
				{Key: "first_name", Value: "text"},
				{Key: "last_name", Value: "text"},
			},
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		slog.Error("Failed to create indexes for customers collection", "error", err)
		return err
	}

	slog.Info("Successfully created indexes for customers collection")
	return nil
}

// UpdateCustomerStopStatus updates the stop field and stopped_at timestamp for a customer
func UpdateCustomerStopStatus(ctx context.Context, customerID, pageID string, stop bool) (*models.Customer, error) {
	db := GetDatabase()
	collection := db.Collection("customers")

	filter := bson.M{
		"customer_id": customerID,
		"page_id":     pageID,
	}

	update := bson.M{
		"$set": bson.M{
			"stop":       stop,
			"updated_at": time.Now(),
		},
	}

	// If setting stop to true, add stopped_at timestamp
	if stop {
		now := time.Now()
		update["$set"].(bson.M)["stopped_at"] = &now
	} else {
		// If setting stop to false, remove stopped_at timestamp
		update["$unset"] = bson.M{"stopped_at": 1}
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var customer models.Customer
	err := collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&customer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Customer not found
		}
		slog.Error("Failed to update customer stop status",
			"customerID", customerID,
			"pageID", pageID,
			"error", err)
		return nil, err
	}

	slog.Info("Customer stop status updated",
		"customerID", customerID,
		"pageID", pageID,
		"stop", stop)

	return &customer, nil
}

// GetStoppedCustomersCount returns the count of customers who want to talk to a real person
func GetStoppedCustomersCount(ctx context.Context, companyID string) (int64, error) {
	db := GetDatabase()
	collection := db.Collection("customers")

	filter := bson.M{
		"company_id": companyID,
		"stop":       true,
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		slog.Error("Failed to count stopped customers",
			"companyID", companyID,
			"error", err)
		return 0, err
	}

	return count, nil
}

// UpdateCustomerAgentName updates the agent_name field for a customer
func UpdateCustomerAgentName(ctx context.Context, customerID, pageID, agentName string) error {
	db := GetDatabase()
	collection := db.Collection("customers")

	filter := bson.M{
		"customer_id": customerID,
		"page_id":     pageID,
	}

	update := bson.M{
		"$set": bson.M{
			"agent_name": agentName,
			"updated_at": time.Now(),
		},
	}

	result, err := collection.UpdateOne(ctx, filter, update)
	if err != nil {
		slog.Error("Failed to update customer agent name",
			"customerID", customerID,
			"pageID", pageID,
			"agentName", agentName,
			"error", err)
		return err
	}

	if result.MatchedCount == 0 {
		slog.Warn("No customer found to update agent name",
			"customerID", customerID,
			"pageID", pageID)
	} else {
		slog.Info("Customer agent name updated",
			"customerID", customerID,
			"pageID", pageID,
			"agentName", agentName)
	}

	return nil
}

// GetStoppedCustomers returns a list of customers who want to talk to a real person
func GetStoppedCustomers(ctx context.Context, companyID string, pageID string, limit, skip int) ([]models.Customer, int64, error) {
	db := GetDatabase()
	collection := db.Collection("customers")

	filter := bson.M{
		"company_id": companyID,
		"stop":       true,
	}

	// If pageID is provided, filter by page as well
	if pageID != "" {
		filter["page_id"] = pageID
	}

	// Get total count
	totalCount, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		slog.Error("Failed to count stopped customers",
			"companyID", companyID,
			"pageID", pageID,
			"error", err)
		return nil, 0, err
	}

	// Get customers with pagination
	findOptions := options.Find().
		SetSort(bson.M{"stopped_at": -1}). // Sort by when they requested help, newest first
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		slog.Error("Failed to get stopped customers",
			"companyID", companyID,
			"pageID", pageID,
			"error", err)
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var customers []models.Customer
	if err := cursor.All(ctx, &customers); err != nil {
		return nil, 0, err
	}

	slog.Debug("Retrieved stopped customers",
		"companyID", companyID,
		"pageID", pageID,
		"count", len(customers),
		"total", totalCount)

	return customers, totalCount, nil
}

// AssignAgentToCustomer assigns an agent to handle a customer
func AssignAgentToCustomer(ctx context.Context, customerID, pageID, agentID, agentEmail, agentName string) (*models.Customer, error) {
	db := GetDatabase()
	collection := db.Collection("customers")

	filter := bson.M{
		"customer_id": customerID,
		"page_id":     pageID,
	}

	now := time.Now()
	update := bson.M{
		"$set": bson.M{
			"agent_id":    agentID,
			"agent_email": agentEmail,
			"agent_name":  agentName,
			"assigned_at": &now,
			"is_assigned": true,
			"updated_at":  now,
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var customer models.Customer
	err := collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&customer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Customer not found
		}
		slog.Error("Failed to assign agent to customer",
			"customerID", customerID,
			"pageID", pageID,
			"agentID", agentID,
			"error", err)
		return nil, err
	}

	slog.Info("Agent assigned to customer",
		"customerID", customerID,
		"pageID", pageID,
		"agentID", agentID,
		"agentEmail", agentEmail,
		"agentName", agentName)

	return &customer, nil
}

// UnassignAgentFromCustomer removes agent assignment from a customer
func UnassignAgentFromCustomer(ctx context.Context, customerID, pageID string) (*models.Customer, error) {
	db := GetDatabase()
	collection := db.Collection("customers")

	filter := bson.M{
		"customer_id": customerID,
		"page_id":     pageID,
	}

	update := bson.M{
		"$set": bson.M{
			"is_assigned": false,
			"updated_at":  time.Now(),
		},
		"$unset": bson.M{
			"agent_id":    1,
			"agent_email": 1,
			"agent_name":  1,
			"assigned_at": 1,
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var customer models.Customer
	err := collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&customer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Customer not found
		}
		slog.Error("Failed to unassign agent from customer",
			"customerID", customerID,
			"pageID", pageID,
			"error", err)
		return nil, err
	}

	slog.Info("Agent unassigned from customer",
		"customerID", customerID,
		"pageID", pageID)

	return &customer, nil
}

// UpdateCustomerAssignmentStatus updates only the is_assigned field for a customer
func UpdateCustomerAssignmentStatus(ctx context.Context, customerID, pageID string, isAssigned bool) (*models.Customer, error) {
	db := GetDatabase()
	collection := db.Collection("customers")

	filter := bson.M{
		"customer_id": customerID,
		"page_id":     pageID,
	}

	update := bson.M{
		"$set": bson.M{
			"is_assigned": isAssigned,
			"updated_at":  time.Now(),
		},
	}

	// If setting to false, also clear agent fields
	if !isAssigned {
		update["$unset"] = bson.M{
			"agent_id":    1,
			"agent_email": 1,
			"agent_name":  1,
			"assigned_at": 1,
		}
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var customer models.Customer
	err := collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&customer)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Customer not found
		}
		slog.Error("Failed to update customer assignment status",
			"customerID", customerID,
			"pageID", pageID,
			"isAssigned", isAssigned,
			"error", err)
		return nil, err
	}

	slog.Info("Customer assignment status updated",
		"customerID", customerID,
		"pageID", pageID,
		"isAssigned", isAssigned)

	return &customer, nil
}
