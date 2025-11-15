# 3D Drone Model Integration Guide

## Step-by-Step Instructions

### 1. Install Dependencies ✅
The required packages have been installed:
- `@react-three/fiber` - React renderer for Three.js
- `@react-three/drei` - Useful helpers for React Three Fiber
- `three` - 3D graphics library

### 2. Where to Place Your .glb File
Place your drone 3D model file here:
```
/client/public/models/drone.glb
```

**Important Notes:**
- The file must be named `drone.glb` (or update the path in `Drone3D.js`)
- The file should be in `.glb` format (GLTF Binary format)
- You can also use `.gltf` format, but `.glb` is recommended for better performance

### 3. File Structure
```
client/
├── public/
│   └── models/
│       └── drone.glb          ← Place your 3D model here
├── src/
│   ├── components/
│   │   ├── Drone3D.js         ← 3D component (already created)
│   │   └── styles/
│   │       └── Drone3D.module.css
│   └── pages/
│       └── DetectionPage.js   ← Integration (already done)
```

### 4. How It Works
- When a drone is detected (`classification.isDrone === true`), the 3D model will appear below the map
- The 3D viewer includes:
  - Interactive orbit controls (drag to rotate, scroll to zoom)
  - Lighting setup (ambient + directional + point lights)
  - Grid helper for reference

### 5. Customizing the Model
If your model file has a different name or path, update it in `DetectionPage.js`:
```javascript
<Drone3D modelPath="/models/your-model-name.glb" showControls={true} />
```

### 6. Model Scale
If your model appears too large or too small, adjust the scale in `Drone3D.js`:
```javascript
scale={1}  // Change this value (e.g., 0.5 for smaller, 2 for larger)
```

### 7. Where to Get 3D Drone Models
- **Sketchfab**: https://sketchfab.com (search for "drone", filter by "Downloadable" and "GLTF")
- **Poly Haven**: https://polyhaven.com/models
- **TurboSquid**: https://www.turbosquid.com (some free models available)
- **Free3D**: https://free3d.com

Make sure to download models in `.glb` or `.gltf` format.

### 8. Testing
1. Start your React app: `npm start`
2. Make a detection that results in `isDrone: true`
3. The 3D model should appear below the map
4. If you see an error, check:
   - File exists at `/public/models/drone.glb`
   - File is a valid GLB/GLTF format
   - Browser console for specific errors

### Troubleshooting
- **Model not loading**: Check browser console for errors, verify file path
- **Model too large/small**: Adjust `scale` prop in `DroneModel` component
- **Model appears dark**: Adjust light intensities in `Drone3D` component
- **Performance issues**: Use `.glb` format instead of `.gltf`, reduce model complexity

