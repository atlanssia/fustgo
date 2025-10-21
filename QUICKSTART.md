# FustGo DataX - Quick Start Guide

Get started with FustGo DataX in 5 minutes!

## What is FustGo DataX?

FustGo DataX is an ETL/ELT data synchronization system that helps you move and transform data between different sources. Think of it as a data pipeline builder.

**Current Version: 0.1.0 (50% Complete)**

### What Works Now ✅

- ✅ CSV file reading and writing
- ✅ Data filtering by conditions
- ✅ Field renaming/mapping
- ✅ Batch processing
- ✅ Progress tracking
- ✅ Docker deployment

### What's Coming Soon ⏳

- Database connectors (MySQL, PostgreSQL)
- REST API
- Web UI
- Job scheduling
- More processors

---

## Installation

### Option 1: Using Docker (Recommended)

```bash
# Clone the repository
git clone https://github.com/atlanssia/fustgo.git
cd fustgo

# Start with Docker Compose
cd deploy/standalone
docker-compose up -d

# Check status
docker-compose ps
docker-compose logs -f fustgo
```

### Option 2: Build from Source

**Requirements:**
- Go 1.23 or higher
- SQLite (embedded, no separate install needed)

```bash
# Clone the repository
git clone https://github.com/atlanssia/fustgo.git
cd fustgo

# Download dependencies
go mod download

# Build
go build -o fustgo ./cmd/fustgo

# Run
./fustgo --version
```

---

## Your First Data Pipeline

Let's create a simple pipeline to filter and transform CSV data.

### Step 1: Prepare Test Data

Create `input.csv`:
```csv
id,name,age,city
1,Alice,25,NYC
2,Bob,17,LA
3,Charlie,30,SF
4,Diana,16,Boston
5,Eve,22,Seattle
```

### Step 2: Create Pipeline Configuration

Create `pipeline.yaml`:
```yaml
# Read from CSV file
input:
  type: csv
  path: ./input.csv
  has_header: true
  delimiter: ","

# Process the data
pipeline:
  processors:
    # Filter: Only keep records where age > 18
    - type: filter
      condition: "age > 18"
      mode: include
    
    # Mapping: Rename fields
    - type: mapping
      field_mappings:
        name: full_name
        city: location

# Write to CSV file
output:
  type: csv
  path: ./output.csv
  write_header: true
  delimiter: ","
```

### Step 3: Run the Pipeline

#### Using Docker:
```bash
# Copy files into container
docker cp input.csv fustgo-standalone:/data/
docker cp pipeline.yaml fustgo-standalone:/app/

# Run pipeline (example - API coming soon)
# For now, this is a demonstration of the config format
```

#### Using Built Binary:
```bash
# Direct execution (programmatic API)
# Web API and CLI execution coming in v0.2.0
```

### Step 4: Check Results

Expected `output.csv`:
```csv
id,full_name,age,location
1,Alice,25,NYC
3,Charlie,30,SF
5,Eve,22,Seattle
```

Notice that:
- ✅ Bob and Diana were filtered out (age ≤ 18)
- ✅ "name" was renamed to "full_name"
- ✅ "city" was renamed to "location"

---

## Configuration Guide

### Input Plugins

#### CSV Input
```yaml
input:
  type: csv
  path: /path/to/file.csv
  has_header: true      # Does file have header row?
  delimiter: ","        # Field delimiter (default: ,)
```

### Processor Plugins

#### Filter Processor
```yaml
processors:
  - type: filter
    condition: "age > 18"          # Simple condition
    mode: include                   # or "exclude"
```

**Supported operators:**
- `=`, `==` - Equals
- `!=` - Not equals
- `>`, `<`, `>=`, `<=` - Comparisons
- `contains` - String contains

**Examples:**
```yaml
# Include only active users
condition: "status = active"

# Exclude test accounts
condition: "email contains test"
mode: exclude

# Age filter
condition: "age >= 21"
```

#### Mapping Processor
```yaml
processors:
  - type: mapping
    field_mappings:
      old_field_name: new_field_name
      user_id: id
      email_address: email
```

### Output Plugins

#### CSV Output
```yaml
output:
  type: csv
  path: /path/to/output.csv
  write_header: true    # Write header row?
  delimiter: ","        # Field delimiter
  append: false         # Append or overwrite?
```

---

## Common Use Cases

### Use Case 1: Data Cleaning

Filter out incomplete records:

```yaml
input:
  type: csv
  path: ./messy_data.csv

pipeline:
  processors:
    - type: filter
      condition: "email"        # Keep only records with email
      mode: include

output:
  type: csv
  path: ./clean_data.csv
```

### Use Case 2: Column Selection

Keep only specific columns by renaming and filtering:

```yaml
input:
  type: csv
  path: ./full_data.csv

pipeline:
  processors:
    - type: mapping
      field_mappings:
        customer_id: id
        customer_name: name
        customer_email: email

output:
  type: csv
  path: ./selected_columns.csv
```

### Use Case 3: Data Standardization

Standardize field names across different sources:

```yaml
input:
  type: csv
  path: ./source_a.csv

pipeline:
  processors:
    - type: mapping
      field_mappings:
        firstName: first_name
        lastName: last_name
        emailAddr: email
        phoneNum: phone

output:
  type: csv
  path: ./standardized.csv
```

---

## Monitoring

### Check Logs

**Docker:**
```bash
docker-compose logs -f fustgo
```

**Binary:**
```bash
tail -f /var/log/fustgo/fustgo.log
```

### View Statistics

Statistics are tracked for each pipeline run:
- Records read
- Records processed
- Records written
- Records filtered
- Execution time

---

## Troubleshooting

### Common Issues

#### File Not Found
```
Error: failed to open file: no such file or directory
```
**Solution:** Check that the file path is correct and accessible.

#### Invalid Configuration
```
Error: validation failed
```
**Solution:** Validate your YAML syntax and required fields.

#### Permission Denied
```
Error: permission denied
```
**Solution:** Ensure the user has read/write permissions for files.

### Getting Help

1. Check the [Documentation](docs/)
2. Review [Plugin Development Guide](docs/PLUGIN_DEVELOPMENT.md)
3. Open an [Issue](https://github.com/atlanssia/fustgo/issues)

---

## Next Steps

### Learn More

- 📖 Read the [Full Documentation](README.md)
- 🔌 Learn [Plugin Development](docs/PLUGIN_DEVELOPMENT.md)
- 🤝 Check [Contributing Guidelines](CONTRIBUTING.md)
- 📊 Review [Project Status](PROJECT_STATUS.md)

### Coming in v0.2.0

- **Job Manager** - Create and manage jobs via API
- **MySQL Plugin** - Read/write to MySQL databases
- **PostgreSQL Plugin** - Read/write to PostgreSQL
- **REST API** - HTTP API for job management
- **Scheduler** - Cron-based job scheduling

### Try Advanced Features

Once comfortable with basics:
1. Combine multiple processors
2. Handle large files with batch processing
3. Set up Docker deployment
4. Create custom plugins

---

## Example: Complete Workflow

Here's a real-world example - cleaning and transforming customer data:

**Input data (`customers.csv`):**
```csv
cust_id,fname,lname,email,age,status
1,John,Doe,john@example.com,25,active
2,Jane,Smith,,30,inactive
3,Bob,Wilson,bob@example.com,17,active
4,Alice,Brown,alice@example.com,28,active
```

**Pipeline (`clean_customers.yaml`):**
```yaml
input:
  type: csv
  path: ./customers.csv
  has_header: true

pipeline:
  processors:
    # Step 1: Filter out records without email
    - type: filter
      condition: "email"
      mode: include
    
    # Step 2: Filter active users only
    - type: filter
      condition: "status = active"
      mode: include
    
    # Step 3: Filter adults only (age >= 18)
    - type: filter
      condition: "age >= 18"
      mode: include
    
    # Step 4: Standardize field names
    - type: mapping
      field_mappings:
        cust_id: customer_id
        fname: first_name
        lname: last_name

output:
  type: csv
  path: ./customers_clean.csv
  write_header: true
```

**Output (`customers_clean.csv`):**
```csv
customer_id,first_name,last_name,email,age,status
1,John,Doe,john@example.com,25,active
4,Alice,Brown,alice@example.com,28,active
```

**Results:**
- ✅ Removed records without email (Jane)
- ✅ Removed inactive users (Jane)
- ✅ Removed underage users (Bob, age 17)
- ✅ Standardized field names
- ✅ Kept only 2 valid active adult customers with email

---

## Performance Tips

1. **Use appropriate batch sizes** (default: 1000 records)
2. **Filter early** - Place filter processors before heavy transformations
3. **Monitor memory** - For large files, process in batches
4. **Test with sample data** - Validate pipeline with small dataset first

---

## Support

- **Documentation:** [docs/](docs/)
- **Issues:** [GitHub Issues](https://github.com/atlanssia/fustgo/issues)
- **Email:** fustgo@example.com

---

**Ready to build data pipelines? Let's go! 🚀**
