import React, { useState, useEffect, useRef } from 'react';
import { FaMicrophone, FaMicrophoneSlash, FaVolumeUp, FaRobot } from 'react-icons/fa';
import styles from './styles/Chatbot.module.css';

const Chatbot = ({ onClose }) => {
  const [messages, setMessages] = useState([]);
  const [isListening, setIsListening] = useState(false);
  const [isProcessing, setIsProcessing] = useState(false);
  const [isSpeaking, setIsSpeaking] = useState(false);
  const [transcript, setTranscript] = useState('');
  const [error, setError] = useState('');
  
  const recognitionRef = useRef(null);
  const messagesEndRef = useRef(null);
  const audioRef = useRef(null);

  useEffect(() => {
    // Initialize Speech Recognition
    if ('webkitSpeechRecognition' in window || 'SpeechRecognition' in window) {
      const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition;
      recognitionRef.current = new SpeechRecognition();
      
      recognitionRef.current.continuous = false;
      recognitionRef.current.interimResults = true;
      recognitionRef.current.lang = 'en-US';
      
      recognitionRef.current.onstart = () => {
        setIsListening(true);
        setError('');
      };
      
      recognitionRef.current.onresult = (event) => {
        let interimTranscript = '';
        let finalTranscript = '';
        
        for (let i = event.resultIndex; i < event.results.length; i++) {
          const transcript = event.results[i][0].transcript;
          if (event.results[i].isFinal) {
            finalTranscript += transcript;
          } else {
            interimTranscript += transcript;
          }
        }
        
        setTranscript(finalTranscript + interimTranscript);
        
        if (finalTranscript) {
          handleUserMessage(finalTranscript);
        }
      };
      
      recognitionRef.current.onerror = (event) => {
        setError(`Speech recognition error: ${event.error}`);
        setIsListening(false);
      };
      
      recognitionRef.current.onend = () => {
        setIsListening(false);
        setTranscript('');
      };
    } else {
      setError('Speech recognition not supported in this browser');
    }

    return () => {
      if (recognitionRef.current) {
        recognitionRef.current.stop();
      }
    };
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  const startListening = () => {
    if (recognitionRef.current && !isListening) {
      try {
        recognitionRef.current.start();
      } catch (error) {
        setError('Failed to start speech recognition');
      }
    }
  };

  const stopListening = () => {
    if (recognitionRef.current && isListening) {
      recognitionRef.current.stop();
    }
  };

  const handleUserMessage = async (message) => {
    const userMessage = { type: 'user', content: message, timestamp: new Date() };
    setMessages(prev => [...prev, userMessage]);
    setIsProcessing(true);

    try {
      // Send message to backend for Gemini processing
      const response = await fetch(`${process.env.REACT_APP_BACKEND_URL}/api/chat`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ message }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }

      const data = await response.json();
      const botMessage = { type: 'bot', content: data.response, timestamp: new Date() };
      setMessages(prev => [...prev, botMessage]);

      // Convert response to speech using Google TTS
      await speakText(data.response);

    } catch (error) {
      console.error('Error processing message:', error);
      const errorMessage = { 
        type: 'bot', 
        content: 'Sorry, I encountered an error processing your message. Please try again.', 
        timestamp: new Date() 
      };
      setMessages(prev => [...prev, errorMessage]);
    } finally {
      setIsProcessing(false);
    }
  };

  const speakText = async (text) => {
    try {
      setIsSpeaking(true);
      
      // Call backend TTS endpoint for Google TTS streaming
      const response = await fetch(`${process.env.REACT_APP_BACKEND_URL}/api/tts`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ text }),
      });

      if (!response.ok) {
        throw new Error(`TTS error! status: ${response.status}`);
      }

      const audioBlob = await response.blob();
      const audioUrl = URL.createObjectURL(audioBlob);
      
      if (audioRef.current) {
        audioRef.current.src = audioUrl;
        audioRef.current.onended = () => {
          setIsSpeaking(false);
          URL.revokeObjectURL(audioUrl);
        };
        await audioRef.current.play();
      }

    } catch (error) {
      console.error('Error with text-to-speech:', error);
      setIsSpeaking(false);
      
      // Fallback to browser's built-in speech synthesis
      if ('speechSynthesis' in window) {
        const utterance = new SpeechSynthesisUtterance(text);
        utterance.onend = () => setIsSpeaking(false);
        speechSynthesis.speak(utterance);
      }
    }
  };

  const clearChat = () => {
    setMessages([]);
  };

  return (
    <div className={styles.chatbot}>
      <div className={styles.header}>
        <FaRobot className={styles.botIcon} />
        <h3>AALIS Assistant</h3>
        <div className={styles.headerButtons}>
          <button onClick={clearChat} className={styles.clearButton}>Clear</button>
          {onClose && (
            <button onClick={onClose} className={styles.closeButton}>×</button>
          )}
        </div>
      </div>
      
      <div className={styles.messagesContainer}>
        {messages.length === 0 && (
          <div className={styles.welcomeMessage}>
            <p>Hi! I'm your AALIS Assistant. Ask me anything about drone detection, acoustic analysis, or system operations.</p>
            <p>Click the microphone to speak or type your message below.</p>
          </div>
        )}
        
        {messages.map((message, index) => (
          <div key={index} className={`${styles.message} ${styles[message.type]}`}>
            <div className={styles.messageContent}>
              {message.content}
            </div>
            <div className={styles.timestamp}>
              {message.timestamp.toLocaleTimeString()}
            </div>
          </div>
        ))}
        
        {isProcessing && (
          <div className={`${styles.message} ${styles.bot}`}>
            <div className={styles.messageContent}>
              <div className={styles.typing}>
                <span></span>
                <span></span>
                <span></span>
              </div>
            </div>
          </div>
        )}
        
        <div ref={messagesEndRef} />
      </div>

      <div className={styles.inputContainer}>
        {transcript && (
          <div className={styles.transcriptPreview}>
            Listening: {transcript}
          </div>
        )}
        
        <div className={styles.controls}>
          <button
            onClick={isListening ? stopListening : startListening}
            className={`${styles.micButton} ${isListening ? styles.listening : ''}`}
            disabled={isProcessing}
          >
            {isListening ? <FaMicrophoneSlash /> : <FaMicrophone />}
          </button>
          
          {isSpeaking && (
            <div className={styles.speakingIndicator}>
              <FaVolumeUp className={styles.speakingIcon} />
              <span>Speaking...</span>
            </div>
          )}
        </div>

        {error && (
          <div className={styles.error}>
            {error}
          </div>
        )}
      </div>

      <audio ref={audioRef} style={{ display: 'none' }} />
    </div>
  );
};

export default Chatbot;