import React, { useState } from 'react';
import { FaRobot, FaTimes } from 'react-icons/fa';
import Chatbot from './Chatbot';
import styles from './styles/ChatbotToggle.module.css';

const ChatbotToggle = () => {
  const [isOpen, setIsOpen] = useState(false);

  const toggleChatbot = () => {
    setIsOpen(!isOpen);
  };

  return (
    <>
      {/* Floating toggle button */}
      {!isOpen && (
        <button 
          className={`${styles.toggleButton} ${isOpen ? styles.active : ''}`}
          onClick={toggleChatbot}
          title={isOpen ? 'Close Assistant' : 'Open AALIS Assistant'}
        >
          {isOpen ? <FaTimes /> : <FaRobot />}
        </button>
      )}

      {/* Chatbot component */}
      {isOpen && (
        <div className={styles.chatbotContainer}>
          <Chatbot onClose={() => setIsOpen(false)} />
        </div>
      )}
    </>
  );
};

export default ChatbotToggle;