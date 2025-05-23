import React, { useEffect, useState } from 'react';
import { WindowSetSize } from '../../../wailsjs/runtime/runtime';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import LLMSettingsComponent from './LLMSettingsComponent';
import MCPSettingsComponent from './MCPSettingsComponent';

function SettingsPage() {
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState('general');

  useEffect(() => {
    WindowSetSize(650, 500); // Set window size for settings page
  }, []);

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 p-4 -webkit-app-region-drag">
        <button 
          onClick={() => navigate('/')}
          className="p-1.5 rounded-md hover:bg-muted/50 text-muted-foreground hover:text-text transition-colors -webkit-app-region-no-drag"
          title="Back to Search"
        >
          <ArrowLeft className="w-4 h-4" />
        </button>
        <h2 className="text-lg font-semibold text-left">Settings</h2>
      </div>

      {/* Tabs */}
      <div className="px-4 border-b border-border -webkit-app-region-no-drag">
        <div className="flex space-x-4">
          <button
            onClick={() => setActiveTab('general')}
            className={`pb-2 text-sm ${activeTab === 'general' ? 'text-primary border-b-2 border-primary font-medium' : 'text-muted-foreground hover:text-text'}`}
          >
            General
          </button>
          <button
            onClick={() => setActiveTab('mcp')}
            className={`pb-2 text-sm ${activeTab === 'mcp' ? 'text-primary border-b-2 border-primary font-medium' : 'text-muted-foreground hover:text-text'}`}
          >
            MCP Servers
          </button>
        </div>
      </div>

      <div className="flex-1 px-4 pb-4 space-y-6 overflow-auto -webkit-app-region-no-drag">
        {/* LLM Settings Tab */}
        {activeTab === 'general' && <LLMSettingsComponent />}
        
        {/* MCP Servers Tab */}
        {activeTab === 'mcp' && <MCPSettingsComponent />}
      </div>
    </div>
  );
}

export default SettingsPage;