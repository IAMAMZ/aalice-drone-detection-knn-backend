#!/usr/bin/env python3
"""
Download PANNS data files for Windows

The panns-inference library tries to use wget, which doesn't exist on Windows.
This script manually downloads the required files using Python's requests library.
"""

import os
import sys
import requests
from pathlib import Path

def download_file(url, dest_path):
    """Download a file with progress indication"""
    print(f"Downloading {os.path.basename(dest_path)}...")
    
    try:
        response = requests.get(url, stream=True)
        response.raise_for_status()
        
        total_size = int(response.headers.get('content-length', 0))
        block_size = 8192
        downloaded = 0
        
        with open(dest_path, 'wb') as f:
            for chunk in response.iter_content(block_size):
                if chunk:
                    f.write(chunk)
                    downloaded += len(chunk)
                    if total_size > 0:
                        percent = (downloaded / total_size) * 100
                        print(f"  Progress: {percent:.1f}%", end='\r')
        
        print(f"\n  ✓ Downloaded to {dest_path}")
        return True
        
    except Exception as e:
        print(f"\n  ✗ Error: {e}")
        return False


def setup_panns_data():
    """Download PANNS model files and labels"""
    
    # Get home directory (works on Windows, Mac, Linux)
    home = Path.home()
    panns_dir = home / "panns_data"
    
    print(f"Setting up PANNS data in: {panns_dir}")
    
    # Create directory if it doesn't exist
    panns_dir.mkdir(exist_ok=True)
    
    # Files to download
    files = {
        'class_labels_indices.csv': 'https://raw.githubusercontent.com/audioset/ontology/master/ontology.json',
        # Actually, let's get the correct CSV file
    }
    
    # Download class labels CSV
    csv_url = "https://raw.githubusercontent.com/qiuqiangkong/audioset_tagging_cnn/master/metadata/class_labels_indices.csv"
    csv_path = panns_dir / "class_labels_indices.csv"
    
    if csv_path.exists():
        print(f"✓ {csv_path.name} already exists")
    else:
        if not download_file(csv_url, csv_path):
            return False
    
    # Download Cnn14 checkpoint (the model weights)
    model_url = "https://zenodo.org/record/3987831/files/Cnn14_mAP%3D0.431.pth?download=1"
    model_path = panns_dir / "Cnn14_mAP=0.431.pth"
    
    if model_path.exists():
        print(f"✓ {model_path.name} already exists (~300MB)")
    else:
        print(f"\nDownloading PANNS model (~300MB, this will take a few minutes)...")
        if not download_file(model_url, model_path):
            return False
    
    print("\n" + "="*60)
    print("✓ PANNS data setup complete!")
    print("="*60)
    print(f"\nFiles installed to: {panns_dir}")
    print(f"  - {csv_path.name}")
    print(f"  - {model_path.name}")
    print("\nYou can now run: python embedding_service.py")
    
    return True


if __name__ == '__main__':
    try:
        success = setup_panns_data()
        sys.exit(0 if success else 1)
    except KeyboardInterrupt:
        print("\n\nDownload cancelled by user")
        sys.exit(1)
    except Exception as e:
        print(f"\n\nError: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

