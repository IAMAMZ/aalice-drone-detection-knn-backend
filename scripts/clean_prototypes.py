#!/usr/bin/env python3
"""
Remove prototypes with zero harmonic features from prototypes.json
This allows you to re-upload them with proper feature extraction.
"""

import json
import argparse

EXPECTED_FEATURE_COUNT = 19

def has_zero_harmonic_features(features):
    """Check if harmonic features (last 3) are zeros."""
    if len(features) != EXPECTED_FEATURE_COUNT:
        return True  # Wrong dimension
    # Check last 3 features (harmonic descriptors)
    harmonic_features = features[-3:]
    return all(f == 0 for f in harmonic_features)

def remove_zero_harmonic_prototypes(input_file, output_file=None, dry_run=False):
    """Remove prototypes that have zeros for harmonic features (last 3 features)."""
    
    if output_file is None:
        output_file = input_file
    
    with open(input_file, 'r') as f:
        prototypes = json.load(f)
    
    original_count = len(prototypes)
    kept = []
    removed = []
    
    for proto in prototypes:
        features = proto.get('features', [])
        
        if has_zero_harmonic_features(features):
            removed.append({
                'id': proto.get('id'),
                'label': proto.get('label'),
                'source': proto.get('source', 'unknown')
            })
        else:
            kept.append(proto)
    
    print(f"Original prototypes: {original_count}")
    print(f"Keeping: {len(kept)}")
    print(f"Removing: {len(removed)}")
    
    if removed:
        print(f"\nPrototypes to be removed (re-upload these):")
        for item in removed:
            print(f"  - {item['id']}: {item['label']} (source: {item['source']})")
    
    if not dry_run:
        with open(output_file, 'w') as f:
            json.dump(kept, f, indent=2)
        print(f"\n✅ Saved cleaned prototypes to {output_file}")
        print(f"   You can now re-upload the removed prototypes via the web interface.")
    else:
        print(f"\n⚠️  DRY RUN - no changes made")
        print(f"   Run without --dry-run to actually remove them")
    
    return kept, removed

if __name__ == '__main__':
    parser = argparse.ArgumentParser(description='Remove prototypes with zero harmonic features')
    parser.add_argument('input', nargs='?', default='server/drone/prototypes.json',
                       help='Input prototypes.json file')
    parser.add_argument('--output', '-o', help='Output file (default: overwrite input)')
    parser.add_argument('--dry-run', action='store_true', help='Show what would be removed without making changes')
    args = parser.parse_args()
    
    remove_zero_harmonic_prototypes(args.input, args.output, args.dry_run)

