#!/bin/bash
##
## Copyright (c) Microsoft Corporation.
## Licensed under the MIT license.
## SPDX-License-Identifier: MIT
##

deployment=$1 # first parameter file is the deployment object
references=$2 # second parmeter file contains the reference components

# the apply script is called with a list of components to be updated via
# the references parameter
components=$(jq -c '.[]' "$references")

echo "COMPONENTS: $components"

while IFS= read -r line; do
    # Extract the name and age fields from each JSON object
    name=$(echo "$line" | jq -r '.name')
    properties=$(echo "$line" | jq -r '.properties')
    echo "NAME: $name"
    echo "PROPERTIES: $properties"
done <<< "$components"

# optionally, you can use the deployment parameter to get additional contextual information as needed.
# for example, you can the following query to get the instance scope. 

scope=$(jq '.instance.scope' "$deployment")
echo "SCOPE: $scope"

# Define the Python Flask server code as an embedded script
MODEL_SERVER_SCRIPT=$(cat <<'EOF'
#!/usr/bin/env python3
import os
import pickle
import numpy as np
from flask import Flask, request, jsonify

# Initialize Flask app
app = Flask(__name__)

# Load the model
MODEL_PATH = os.path.join(os.getcwd(), 'iris_model.pkl')

print(f"Loading model from {MODEL_PATH}")
with open(MODEL_PATH, 'rb') as f:
    model = pickle.load(f)

# Define class names for Iris dataset
IRIS_CLASSES = ['setosa', 'versicolor', 'virginica']

@app.route('/')
def home():
    """Home page with usage instructions"""
    return """
    <h1>Iris Model Prediction API</h1>
    <p>Send a POST request to /predict with JSON data containing Iris features.</p>
    <h2>Example:</h2>
    <pre>
    curl -X POST http://localhost:5000/predict \\
      -H "Content-Type: application/json" \\
      -d '{"features": [5.1, 3.5, 1.4, 0.2]}'
    </pre>
    """

@app.route('/predict', methods=['POST'])
def predict():
    """Endpoint for making predictions"""
    try:
        # Get features from request
        data = request.get_json()
        
        if not data or 'features' not in data:
            return jsonify({
                'error': 'Invalid request. Please provide features in JSON format.',
                'example': {'features': [5.1, 3.5, 1.4, 0.2]}
            }), 400
        
        features = data['features']
        
        # Validate input
        if not isinstance(features, list) or len(features) != 4:
            return jsonify({
                'error': 'Features must be a list of 4 numeric values',
                'example': {'features': [5.1, 3.5, 1.4, 0.2]}
            }), 400
        
        # Convert to numpy array
        features_array = np.array([features])
        
        # Make prediction
        prediction = model.predict(features_array)[0]
        probabilities = model.predict_proba(features_array)[0]
        
        # Format response
        response = {
            'prediction': {
                'class_id': int(prediction),
                'class_name': IRIS_CLASSES[prediction]
            },
            'probabilities': {
                IRIS_CLASSES[i]: float(prob) for i, prob in enumerate(probabilities)
            },
            'input_features': {
                'sepal_length': features[0],
                'sepal_width': features[1],
                'petal_length': features[2],
                'petal_width': features[3]
            }
        }
        
        return jsonify(response)
    
    except Exception as e:
        return jsonify({'error': str(e)}), 500

@app.route('/info', methods=['GET'])
def model_info():
    """Return information about the model"""
    # Extract some model information
    n_estimators = model.n_estimators
    feature_importances = model.feature_importances_.tolist()
    feature_names = ['sepal_length', 'sepal_width', 'petal_length', 'petal_width']
    
    return jsonify({
        'model_type': model.__class__.__name__,
        'parameters': {
            'n_estimators': n_estimators
        },
        'feature_importances': {
            feature_names[i]: importance for i, importance in enumerate(feature_importances)
        },
        'classes': IRIS_CLASSES
    })

if __name__ == '__main__':
    print("Starting Iris prediction API server...")
    print("Server is running at http://localhost:5000")
    app.run(host='0.0.0.0', port=5000)
EOF
)

# Create a Python script file with a fixed name
MODEL_SERVER_FILE="${workspace_dir}/modelServe.py"
echo "$MODEL_SERVER_SCRIPT" > "$MODEL_SERVER_FILE"
chmod +x "$MODEL_SERVER_FILE"

# Check if the Python model endpoint is running
model_running=false

# First try a direct HTTP check which is most reliable
if curl -s http://localhost:5000/info > /dev/null; then
    echo "Model endpoint is already running and responding to requests."
    model_running=true
else
    echo "Model endpoint is not running. Starting it..."
    # Get the path to the model
    script_dir=$(realpath "$(dirname "${BASH_SOURCE[0]}")")
    workspace_dir=$(realpath "$script_dir/../../../..")
    
    # Start the model server in the background
    cd "$workspace_dir" || exit 1
    
    # Run the server
    nohup python3 "$MODEL_SERVER_FILE" > model_server.log 2>&1 &
    
    # Wait a moment for the server to start
    sleep 3
    
    # Check if server started successfully with multiple detection methods
    if curl -s http://localhost:5000/info > /dev/null; then
        echo "Model endpoint started and responding to requests."
        model_running=true
    else
        echo "Failed to start model endpoint."
        model_running=false
    fi
fi

# your script needs to generate an output file that contains a map of component results. For each
# component result, the status code should be one of
# 8001: fialed to update
# 8002: failed to delete
# 8003: failed to validate component artifact
# 8004: updated (success)
# 8005: deleted (success)
# 9998: untouched - no actions are taken/necessary

# Set the output status based on whether the model endpoint is running
if [ "$model_running" = true ]; then
    # Test the model endpoint
    if curl -s http://localhost:5000/info > /dev/null; then
        output_results='{
            "com1": {
                "status": 8004,
                "message": "Model endpoint running and responsive"
            }
        }'
    else
        output_results='{
            "com1": {
                "status": 8001,
                "message": "Model endpoint started but not responding"
            }
        }'
    fi
else
    output_results='{
        "com1": {
            "status": 8001,
            "message": "Failed to start model endpoint"
        }
    }'
fi

echo "$output_results" > ${deployment%.*}-output.${deployment##*.}
