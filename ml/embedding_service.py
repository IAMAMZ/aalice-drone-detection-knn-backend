#!/usr/bin/env python3
"""
PANNS Embedding Service for Drone Audio Classification

This service generates deep learning embeddings using PANNS (Pretrained Audio Neural Networks)
for more discriminative drone audio features.

Install:
    pip install panns-inference torch librosa flask
"""

import os
import sys
import numpy as np
import librosa
import torch
from flask import Flask, request, jsonify
from panns_inference import AudioTagging
import tempfile
import logging

app = Flask(__name__)
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Initialize PANNS model globally (loads once at startup)
device = 'cuda' if torch.cuda.is_available() else 'cpu'
logger.info(f"Initializing PANNS on device: {device}")

try:
    at = AudioTagging(
        checkpoint_path=None,  # Downloads automatically first time
        device=device
    )
    logger.info("PANNS model loaded successfully")
except Exception as e:
    logger.error(f"Failed to load PANNS model: {e}")
    at = None


def embed_audio_panns(audio_path):
    """
    Generate PANNS embedding from audio file
    
    Args:
        audio_path: Path to audio file (WAV, MP3, etc.)
        
    Returns:
        numpy array of shape (2048,) containing the embedding
    """
    if at is None:
        raise RuntimeError("PANNS model not loaded")
    
    # Load audio with librosa (handles multiple formats)
    audio, sr = librosa.load(audio_path, sr=32000, mono=True)
    
    # Add batch dimension
    audio = audio[np.newaxis, :]
    
    # Get embeddings (returns tuple of clipwise_output and embedding)
    _, embedding = at.inference(audio)
    
    return embedding[0]  # Return first (and only) embedding


@app.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'model': 'PANNS',
        'device': device,
        'embedding_dim': 2048
    })


@app.route('/embed', methods=['POST'])
def embed():
    """
    Generate embedding for uploaded audio
    
    Request:
        - File upload via multipart/form-data with key 'audio'
        OR
        - JSON with base64 encoded audio in 'audio_data' field
        
    Response:
        JSON with 'embedding' field containing list of 2048 floats
    """
    try:
        # Handle file upload
        if 'audio' in request.files:
            audio_file = request.files['audio']
            
            # Save to temporary file
            with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as tmp:
                audio_file.save(tmp.name)
                tmp_path = tmp.name
            
            try:
                embedding = embed_audio_panns(tmp_path)
                return jsonify({
                    'embedding': embedding.tolist(),
                    'dimension': len(embedding)
                })
            finally:
                os.unlink(tmp_path)
        
        # Handle base64 audio data
        elif request.json and 'audio_data' in request.json:
            import base64
            audio_data = base64.b64decode(request.json['audio_data'])
            
            with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as tmp:
                tmp.write(audio_data)
                tmp_path = tmp.name
            
            try:
                embedding = embed_audio_panns(tmp_path)
                return jsonify({
                    'embedding': embedding.tolist(),
                    'dimension': len(embedding)
                })
            finally:
                os.unlink(tmp_path)
        
        else:
            return jsonify({'error': 'No audio file or audio_data provided'}), 400
            
    except Exception as e:
        logger.error(f"Embedding generation failed: {e}", exc_info=True)
        return jsonify({'error': str(e)}), 500


@app.route('/embed/batch', methods=['POST'])
def embed_batch():
    """
    Generate embeddings for multiple audio files
    
    Request:
        Multiple files via multipart/form-data
        
    Response:
        JSON with 'embeddings' array
    """
    try:
        files = request.files.getlist('audio')
        if not files:
            return jsonify({'error': 'No audio files provided'}), 400
        
        embeddings = []
        for audio_file in files:
            with tempfile.NamedTemporaryFile(suffix='.wav', delete=False) as tmp:
                audio_file.save(tmp.name)
                tmp_path = tmp.name
            
            try:
                embedding = embed_audio_panns(tmp_path)
                embeddings.append({
                    'filename': audio_file.filename,
                    'embedding': embedding.tolist()
                })
            finally:
                os.unlink(tmp_path)
        
        return jsonify({
            'embeddings': embeddings,
            'count': len(embeddings)
        })
        
    except Exception as e:
        logger.error(f"Batch embedding generation failed: {e}", exc_info=True)
        return jsonify({'error': str(e)}), 500


if __name__ == '__main__':
    if at is None:
        logger.error("PANNS model failed to load. Exiting.")
        sys.exit(1)
    
    port = int(os.environ.get('EMBEDDING_SERVICE_PORT', 5002))
    logger.info(f"Starting PANNS embedding service on port {port}")
    app.run(host='0.0.0.0', port=port, debug=False)

