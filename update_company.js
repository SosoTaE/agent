// MongoDB script to update company API key to TEST_MODE
// Run this in MongoDB shell or Compass

db.companies.updateOne(
  { "company_id": "company_economist_georgia_1754827518" },
  { 
    $set: { 
      "claude_api_key": "TEST_MODE"
    } 
  }
);

// Or update by page ID
db.companies.updateOne(
  { "pages.page_id": "102222849213007" },
  { 
    $set: { 
      "claude_api_key": "TEST_MODE"
    } 
  }
);

// Check the update
db.companies.findOne({ "pages.page_id": "102222849213007" });