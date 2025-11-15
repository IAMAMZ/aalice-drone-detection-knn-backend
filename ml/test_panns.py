#!/usr/bin/env python3
"""
Quick test script for PANNS embedding service

Usage:
    python test_panns.py ../Drone-Training-Data/drone_A/A_01.wav
"""

import sys
import numpy as np
from embedding_service import embed_audio_panns, at

def test_embedding(audio_path):
    """Test PANNS embedding generation"""
    print(f"Testing PANNS embedding on: {audio_path}")
    print()
    
    # Check model loaded
    if at is None:
        print("❌ PANNS model not loaded!")
        return False
    
    print("✓ PANNS model loaded")
    print(f"✓ Device: {at.device}")
    print()
    
    # Generate embedding
    print("Generating embedding...")
    try:
        embedding = embed_audio_panns(audio_path)
        print("✓ Embedding generated successfully")
        print()
        
        # Print statistics
        print(f"Embedding dimension: {len(embedding)}")
        print(f"Mean: {np.mean(embedding):.4f}")
        print(f"Std:  {np.std(embedding):.4f}")
        print(f"Min:  {np.min(embedding):.4f}")
        print(f"Max:  {np.max(embedding):.4f}")
        print(f"L2 norm: {np.linalg.norm(embedding):.4f}")
        print()
        
        # Show first 10 values
        print("First 10 values:")
        print([f"{x:.4f}" for x in embedding[:10]])
        print()
        
        return True
        
    except Exception as e:
        print(f"❌ Error: {e}")
        import traceback
        traceback.print_exc()
        return False


def compare_embeddings(audio1, audio2):
    """Compare embeddings from two audio files"""
    print(f"Comparing embeddings:")
    print(f"  File 1: {audio1}")
    print(f"  File 2: {audio2}")
    print()
    
    try:
        emb1 = embed_audio_panns(audio1)
        emb2 = embed_audio_panns(audio2)
        
        # Calculate cosine similarity
        dot = np.dot(emb1, emb2)
        norm1 = np.linalg.norm(emb1)
        norm2 = np.linalg.norm(emb2)
        similarity = dot / (norm1 * norm2)
        
        print(f"Cosine similarity: {similarity:.4f}")
        
        if similarity > 0.95:
            print("→ Very similar (same drone type)")
        elif similarity > 0.85:
            print("→ Similar (possibly same drone)")
        elif similarity > 0.70:
            print("→ Somewhat similar (related drones)")
        else:
            print("→ Different (distinct drones/sounds)")
        
        return True
        
    except Exception as e:
        print(f"❌ Error: {e}")
        return False


if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: python test_panns.py <audio_file> [audio_file2]")
        sys.exit(1)
    
    audio1 = sys.argv[1]
    
    if len(sys.argv) >= 3:
        # Compare two files
        audio2 = sys.argv[2]
        success = compare_embeddings(audio1, audio2)
    else:
        # Test single file
        success = test_embedding(audio1)
    
    sys.exit(0 if success else 1)

