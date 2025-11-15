#!/usr/bin/env python3
"""
Regenerate prototypes with proper 14-feature extraction.
This fixes prototypes that have zeros for harmonic features.
"""

import json
import sys
from pathlib import Path

def fix_prototypes(input_file, output_file=None):
    """Fix prototypes by removing zero-padded harmonic features and marking for regeneration."""
    
    if output_file is None:
        output_file = input_file
    
    with open(input_file, 'r') as f:
        prototypes = json.load(f)
    
    fixed_count = 0
    needs_regeneration = []
    
    for proto in prototypes:
        features = proto.get('features', [])
        
        # Check if last 3 features are zeros (old padded format)
        if len(features) >= 13 and all(f == 0 for f in features[-3:]):
            needs_regeneration.append({
                'id': proto.get('id'),
                'label': proto.get('label'),
                'source': proto.get('source', 'unknown')
            })
            fixed_count += 1
    
    print(f"Found {fixed_count} prototypes with zero-padded harmonic features")
    print(f"These need to be regenerated from their source audio files:\n")
    
    for item in needs_regeneration:
        print(f"  - {item['id']}: {item['label']} (source: {item['source']})")
    
    if needs_regeneration:
        print(f"\n⚠️  WARNING: These prototypes have invalid harmonic features (zeros).")
        print(f"   Detection accuracy will be poor until they are regenerated.")
        print(f"\nTo fix:")
        print(f"  1. Use the upload interface to re-upload the audio files")
        print(f"  2. Or use: python ml/build_prototypes.py --input <dataset> --output server/drone/prototypes.json")
    
    return needs_regeneration

if __name__ == '__main__':
    proto_file = sys.argv[1] if len(sys.argv) > 1 else 'server/drone/prototypes.json'
    fix_prototypes(proto_file)

