import React, { Suspense, useRef, useEffect } from 'react';
import { createRoot } from 'react-dom/client';
import { Canvas, useFrame } from '@react-three/fiber';
import { useGLTF, OrbitControls, PerspectiveCamera } from '@react-three/drei';

function DroneModel3D({ modelPath }) {
  const { scene } = useGLTF(modelPath);
  const droneRef = useRef();

  useEffect(() => {
    if (scene) {
      scene.traverse((child) => {
        if (child.isMesh) {
          child.castShadow = true;
          child.receiveShadow = true;
        }
      });
    }
  }, [scene]);

  // Auto-rotate the drone
  useFrame((state, delta) => {
    if (droneRef.current) {
      droneRef.current.rotation.y += delta * 0.3;
    }
  });

  return (
    <primitive 
      ref={droneRef}
      object={scene} 
      scale={1.2}
      position={[0, 0, 0]}
      rotation={[0, 0, 0]}
    />
  );
}

function DroneMarker3D({ modelPath = '/models/drone.glb', interactive = false }) {
  return (
    <div style={{ 
      width: '100%', 
      height: '100%', 
      position: 'relative', 
      display: 'flex', 
      alignItems: 'center', 
      justifyContent: 'center' 
    }}>
      <Canvas
        camera={{ position: [0, 0, 4], fov: 50 }}
        gl={{ antialias: true, alpha: true }}
        style={{ width: '100%', height: '100%', background: 'transparent' }}
      >
        <ambientLight intensity={0.7} />
        <directionalLight position={[5, 5, 5]} intensity={1} castShadow />
        <pointLight position={[-5, -5, -5]} intensity={0.5} />
        <spotLight position={[0, 10, 0]} intensity={0.5} angle={0.3} penumbra={1} />
        
        {interactive && <OrbitControls enableZoom={true} enablePan={false} />}
        
        <Suspense fallback={null}>
          <DroneModel3D modelPath={modelPath} />
        </Suspense>
      </Canvas>
    </div>
  );
}

export function createDroneMarker3D(modelPath = '/models/drone.glb', interactive = true) {
  const container = document.createElement('div');
  container.style.width = '100%';
  container.style.height = '100%';
  container.style.cursor = interactive ? 'grab' : 'pointer';
  container.style.position = 'relative';
  container.style.display = 'flex';
  container.style.alignItems = 'center';
  container.style.justifyContent = 'center';
  
  const root = createRoot(container);
  root.render(<DroneMarker3D modelPath={modelPath} interactive={interactive} />);
  
  // Store root on container for cleanup
  container._reactRoot = root;
  
  return container;
}

export default DroneMarker3D;