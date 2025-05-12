#!/usr/bin/env python3
"""
A simple Flask API to serve predictions from the trained Iris model.
This script loads the scikit-learn model saved as a pickle file and
creates an endpoint for making predictions.

Dependencies:
- flask
- numpy
- scikit-learn

To install dependencies:
pip install flask numpy scikit-learn

Usage:
1. Run this script: python serve_model.py
2. Send POST requests to http://localhost:5000/predict with JSON data

Example request using curl:
curl -X POST http://localhost:5000/predict \
  -H "Content-Type: application/json" \
  -d '{"features": [5.1, 3.5, 1.4, 0.2]}'

"""

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
    """
    Endpoint for making predictions
    Expects JSON with "features" key containing 4 values
    Returns predicted class and probabilities
    """
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
                'error': 'Features must be a list of 4 numeric values (sepal length, sepal width, petal length, petal width)',
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
    print("API endpoints:")
    print("  - / (GET): Home page with instructions")
    print("  - /predict (POST): Make predictions")
    print("  - /info (GET): Get model information")
    print("\nServer is running at http://localhost:5000")
    app.run(host='0.0.0.0', port=5000, debug=True)
