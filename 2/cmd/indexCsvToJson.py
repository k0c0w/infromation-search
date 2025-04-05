import csv
import json

# Function to convert CSV to JSON
def csv_to_json(csv_file_path, json_file_path):
    data = []  # List to store rows as dictionaries
    
    # Read the CSV file
    with open(csv_file_path, mode='r', encoding='utf-8') as csv_file:
        csv_reader = csv.DictReader(csv_file)  # Automatically maps rows to dictionaries
        for row in csv_reader:
            data.append(row)
    
    # Write the JSON file
    with open(json_file_path, mode='w', encoding='utf-8') as json_file:
        json.dump(data, json_file, ensure_ascii=False, indent=4)  # Indent for readability
    
    print(f"Data has been successfully saved to {json_file_path}")

# Example usage
csv_file_path = 'C:\\CustomDesktop\\informations search\\1\\output\\index.txt'  # Replace with your CSV file path
json_file_path = 'C:\\CustomDesktop\\informations search\\1\\output\\index.json'  # Replace with your desired JSON output file path
csv_to_json(csv_file_path, json_file_path)
