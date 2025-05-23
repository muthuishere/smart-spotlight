import React, { useState, useEffect } from 'react';
import { UpdateSettings, GetSettings, TestAPIConnection } from '../../../wailsjs/go/backend/App';

function LLMSettingsComponent() {
  const [settings, setSettings] = useState({
    baseUrl: '',
    apiKey: '',
    model: '',
    availableModels: [],
  });
  const [isLoading, setIsLoading] = useState(true);
  const [isTesting, setIsTesting] = useState(false);
  const [testError, setTestError] = useState('');
  const [testSuccess, setTestSuccess] = useState(false);
  const [saveSuccess, setSaveSuccess] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      const currentSettings = await GetSettings();
      if (currentSettings) {
        setSettings(currentSettings);
      }
      setIsLoading(false);
    } catch (err) {
      console.error("Error fetching LLM settings:", err);
      setIsLoading(false);
    }
  };

  const handleInputChange = (event) => {
    const { name, value } = event.target;
    setSettings(prevSettings => ({
      ...prevSettings,
      [name]: value
    }));
    setTestError('');
    setTestSuccess(false);
    setSaveSuccess(false);
  };

  const handleTestConnection = async () => {
    setIsTesting(true);
    setTestError('');
    setTestSuccess(false);
    
    try {
      await UpdateSettings(settings);
      await TestAPIConnection();
      setTestSuccess(true);
    } catch (error) {
      setTestError(error.toString());
    } finally {
      setIsTesting(false);
    }
  };

  const handleSaveChanges = async () => {
    setIsSaving(true);
    setSaveSuccess(false);
    try {
      await UpdateSettings(settings);
      setSaveSuccess(true);
      setTimeout(() => {
        setSaveSuccess(false);
      }, 3000);
    } catch (error) {
      console.error('Failed to save settings:', error);
      setTestError('Failed to save settings: ' + error.toString());
    } finally {
      setIsSaving(false);
    }
  };

  if (isLoading) {
    return <div className="p-4 text-center">Loading LLM settings...</div>;
  }

  return (
    <section className="space-y-3 pt-4">
      <h3 className="text-sm font-medium text-left">AI Configuration</h3>
      <div className="space-y-3">
        <div className="flex items-center justify-between gap-4">
          <label className="text-xs text-muted-foreground" htmlFor="baseUrl">Base URL</label>
          <input 
            type="text" 
            id="baseUrl" 
            name="baseUrl"
            value={settings.baseUrl}
            onChange={handleInputChange}
            className="flex-1 max-w-[300px] text-sm bg-background rounded-md border border-border px-2 py-1.5"
            placeholder="https://api.openai.com/v1"
          />
        </div>

        <div className="flex items-center justify-between gap-4">
          <label className="text-xs text-muted-foreground" htmlFor="apiKey">API Key</label>
          <input 
            type="password" 
            id="apiKey" 
            name="apiKey"
            value={settings.apiKey}
            onChange={handleInputChange}
            className="flex-1 max-w-[300px] text-sm bg-background rounded-md border border-border px-2 py-1.5"
            placeholder="Enter your API key"
          />
        </div>

        <div className="flex items-center justify-between gap-4">
          <label className="text-xs text-muted-foreground" htmlFor="model">AI Model</label>
          <input 
            type="text" 
            id="model" 
            name="model"
            value={settings.model}
            onChange={handleInputChange}
            className="flex-1 max-w-[300px] text-sm bg-background rounded-md border border-border px-2 py-1.5"
            placeholder="e.g., gpt-3.5-turbo"
          />
        </div>

        <div className="flex items-center justify-end gap-2">
          <button
            onClick={handleTestConnection}
            disabled={isTesting || !settings.apiKey || !settings.model || !settings.baseUrl}
            className="px-3 py-1 text-sm rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isTesting ? 'Testing...' : 'Test Connection'}
          </button>
        </div>
      </div>
      
      <div className="flex flex-col items-end gap-2 pt-4">
        {saveSuccess && (
          <div className="text-sm text-green-500 animate-fade-in">
            Settings saved successfully!
          </div>
        )}
        {testSuccess && !saveSuccess && (
          <div className="text-sm text-green-500">
            Connection test successful!
          </div>
        )}
        {testError && (
          <div className="text-sm text-red-500">
            {testError}
          </div>
        )}
        <button 
          onClick={handleSaveChanges}
          disabled={isSaving}
          className="px-4 py-1.5 text-sm rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isSaving ? 'Saving...' : 'Save Changes'}
        </button>
      </div>
    </section>
  );
}

export default LLMSettingsComponent;