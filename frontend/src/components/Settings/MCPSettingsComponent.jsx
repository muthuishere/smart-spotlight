import React, { useState, useEffect } from 'react';
import { PlusCircle, Edit, Trash2, ToggleLeft, ToggleRight } from 'lucide-react';
import { 
  GetMCPServers, 
  AddMCPSTDIOServer, 
  AddMCPSSEServer, 
  DeleteMCPServer, 
  EnableMCPServer, 
  DisableMCPServer, 
  SetMCPServerEnabled, 
  UpdateMCPSTDIOServer, 
  UpdateMCPSSEServer 
} from '../../../wailsjs/go/backend/App';

function MCPSettingsComponent() {
  const [mcpServers, setMcpServers] = useState([]);
  const [selectedServer, setSelectedServer] = useState(null);
  const [isAddingServer, setIsAddingServer] = useState(false);
  const [isEditingServer, setIsEditingServer] = useState(false);
  const [serverForm, setServerForm] = useState({
    name: '',
    type: 'stdio',
    command: '',
    args: '',
    env: '',
    url: '',
    headers: '',
    enabled: true, // Default new servers to be enabled
  });
  const [mcpActionStatus, setMcpActionStatus] = useState({
    loading: false,
    success: false,
    error: '',
  });
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    loadMCPServers();
  }, []);

  const loadMCPServers = async () => {
    try {
      const servers = await GetMCPServers();
      if (servers) {
        setMcpServers(servers);
      }
      setIsLoading(false);
    } catch (err) {
      console.error("Error fetching MCP servers:", err);
      setIsLoading(false);
    }
  };

  const handleServerFormChange = (event) => {
    const { name, value, type, checked } = event.target;
    setServerForm(prev => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value
    }));
  };

  const resetServerForm = () => {
    setServerForm({
      name: '',
      type: 'stdio',
      command: '',
      args: '',
      env: '',
      url: '',
      headers: '',
      enabled: true, // Default new servers to be enabled
    });
  };

  const prepareServerForEdit = (server) => {
    let preparedForm = {
      name: server.name,
      type: server.type,
      enabled: server.enabled,
    };

    if (server.type === 'stdio') {
      preparedForm = {
        ...preparedForm,
        command: server.config.command || '',
        args: Array.isArray(server.config.args) ? server.config.args.join(' ') : '',
        env: server.config.env ? 
          Object.entries(server.config.env)
            .map(([key, value]) => `${key}=${value}`)
            .join('\n') : '',
      };
    } else if (server.type === 'sse') {
      preparedForm = {
        ...preparedForm,
        url: server.config.url || '',
        headers: Array.isArray(server.config.headers) ? server.config.headers.join('\n') : '',
      };
    }

    setServerForm(preparedForm);
  };

  const handleAddServer = () => {
    setIsAddingServer(true);
    setIsEditingServer(false);
    resetServerForm();
  };

  const handleEditServer = (server) => {
    setIsAddingServer(false);
    setIsEditingServer(true);
    prepareServerForEdit(server);
  };

  const handleCancelServerAction = () => {
    setIsAddingServer(false);
    setIsEditingServer(false);
    resetServerForm();
  };

  const parseArgsAndEnv = () => {
    // Parse arguments from space-separated string to array
    const args = serverForm.args.trim() ? serverForm.args.split(/\s+/) : [];
    
    // Parse environment variables from newline-separated key=value pairs
    const envLines = serverForm.env.split('\n').filter(line => line.trim());
    const env = {};
    envLines.forEach(line => {
      const [key, ...valueParts] = line.split('=');
      if (key && valueParts.length) {
        env[key.trim()] = valueParts.join('=').trim();
      }
    });
    
    return { args, env };
  };

  const parseHeaders = () => {
    // Parse headers from newline-separated strings
    return serverForm.headers
      .split('\n')
      .filter(header => header.trim())
      .map(header => header.trim());
  };

  const handleSaveServer = async () => {
    setMcpActionStatus({
      loading: true,
      success: false,
      error: '',
    });

    try {
      if (serverForm.type === 'stdio') {
        const { args, env } = parseArgsAndEnv();
        
        if (isEditingServer) {
          await UpdateMCPSTDIOServer(serverForm.name, serverForm.command, args, env);
          // Update enabled state if editing
          await SetMCPServerEnabled(serverForm.name, serverForm.enabled);
        } else {
          await AddMCPSTDIOServer(serverForm.name, serverForm.command, args, env);
          // Set enabled state for new server
          await SetMCPServerEnabled(serverForm.name, serverForm.enabled);
        }
      } else {
        const headers = parseHeaders();
        
        if (isEditingServer) {
          await UpdateMCPSSEServer(serverForm.name, serverForm.url, headers);
          // Update enabled state if editing
          await SetMCPServerEnabled(serverForm.name, serverForm.enabled);
        } else {
          await AddMCPSSEServer(serverForm.name, serverForm.url, headers);
          // Set enabled state for new server
          await SetMCPServerEnabled(serverForm.name, serverForm.enabled);
        }
      }
      
      // Refresh the server list
      const updatedServers = await GetMCPServers();
      setMcpServers(updatedServers);
      
      setMcpActionStatus({
        loading: false,
        success: true,
        error: '',
      });
      
      // Reset form and status after delay
      setTimeout(() => {
        setIsAddingServer(false);
        setIsEditingServer(false);
        resetServerForm();
        setMcpActionStatus(prev => ({
          ...prev,
          success: false,
        }));
      }, 1500);
      
    } catch (error) {
      console.error('Failed to save server:', error);
      setMcpActionStatus({
        loading: false,
        success: false,
        error: `Failed to save server: ${error.toString()}`,
      });
    }
  };

  const handleDeleteServer = async (serverName) => {
    if (!confirm(`Are you sure you want to delete the server "${serverName}"?`)) {
      return;
    }

    setMcpActionStatus({
      loading: true,
      success: false,
      error: '',
    });

    try {
      await DeleteMCPServer(serverName);
      
      // Refresh the server list
      const updatedServers = await GetMCPServers();
      setMcpServers(updatedServers);
      
      // If the deleted server was selected, clear the selection
      if (selectedServer && selectedServer.name === serverName) {
        setSelectedServer(null);
      }
      
      setMcpActionStatus({
        loading: false,
        success: true,
        error: '',
      });
      
      // Reset status after delay
      setTimeout(() => {
        setMcpActionStatus(prev => ({
          ...prev,
          success: false,
        }));
      }, 1500);
      
    } catch (error) {
      console.error('Failed to delete server:', error);
      setMcpActionStatus({
        loading: false,
        success: false,
        error: `Failed to delete server: ${error.toString()}`,
      });
    }
  };

  const handleToggleServerActive = async (serverName, isActive) => {
    setMcpActionStatus({
      loading: true,
      success: false,
      error: '',
    });

    try {
      if (isActive) {
        await DisableMCPServer(serverName);
      } else {
        await EnableMCPServer(serverName);
      }
      
      // Refresh the server list
      const updatedServers = await GetMCPServers();
      setMcpServers(updatedServers);
      
      // Update selected server if needed
      if (selectedServer && selectedServer.name === serverName) {
        const updated = updatedServers.find(s => s.name === serverName);
        if (updated) {
          setSelectedServer(updated);
        }
      }
      
      setMcpActionStatus({
        loading: false,
        success: true,
        error: '',
      });
      
      // Reset status after delay
      setTimeout(() => {
        setMcpActionStatus(prev => ({
          ...prev,
          success: false,
        }));
      }, 1500);
      
    } catch (error) {
      console.error('Failed to toggle server status:', error);
      setMcpActionStatus({
        loading: false,
        success: false,
        error: `Failed to toggle server status: ${error.toString()}`,
      });
    }
  };

  const handleToggleServerEnabled = async (serverName, isEnabled) => {
    setMcpActionStatus({
      loading: true,
      success: false,
      error: '',
    });

    try {
      await SetMCPServerEnabled(serverName, !isEnabled);
      
      // Refresh the server list
      const updatedServers = await GetMCPServers();
      setMcpServers(updatedServers);
      
      // Update selected server if needed
      if (selectedServer && selectedServer.name === serverName) {
        const updated = updatedServers.find(s => s.name === serverName);
        if (updated) {
          setSelectedServer(updated);
        }
      }
      
      setMcpActionStatus({
        loading: false,
        success: true,
        error: '',
      });
      
      // Reset status after delay
      setTimeout(() => {
        setMcpActionStatus(prev => ({
          ...prev,
          success: false,
        }));
      }, 1500);
      
    } catch (error) {
      console.error('Failed to toggle server enabled state:', error);
      setMcpActionStatus({
        loading: false,
        success: false,
        error: `Failed to toggle server enabled state: ${error.toString()}`,
      });
    }
  };

  const renderServerDetails = (server) => {
    if (!server) return null;

    return (
      <div className="mt-4 border border-border rounded-md p-3 space-y-2">
        <div className="flex justify-between items-center">
          <h4 className="font-medium text-sm">{server.name}</h4>
          <div className="flex items-center gap-2">
            <button 
              onClick={() => handleEditServer(server)}
              className="p-1 text-muted-foreground hover:text-text rounded-md hover:bg-muted/50"
              title="Edit Server"
            >
              <Edit size={16} />
            </button>
            <button 
              onClick={() => handleDeleteServer(server.name)}
              className="p-1 text-muted-foreground hover:text-red-500 rounded-md hover:bg-muted/50"
              title="Delete Server"
            >
              <Trash2 size={16} />
            </button>
            <button 
              onClick={() => handleToggleServerActive(server.name, server.isActive)}
              className={`p-1 rounded-md hover:bg-muted/50 ${server.isActive ? 'text-green-500 hover:text-green-600' : 'text-muted-foreground hover:text-text'}`}
              title={server.isActive ? 'Disable Server' : 'Enable Server'}
            >
              {server.isActive ? <ToggleRight size={16} /> : <ToggleLeft size={16} />}
            </button>
          </div>
        </div>
        
        <div className="text-xs text-muted-foreground flex flex-col gap-1">
          <div>Type: <span className="text-text">{server.type}</span></div>
          <div>Status: 
            <span className={`ml-1 ${server.isActive ? 'text-green-500' : 'text-amber-500'}`}>
              {server.isActive ? 'Active' : 'Inactive'}
            </span>
            <span className={`ml-1 ${server.enabled ? 'text-green-500' : 'text-red-500'}`}>
              ({server.enabled ? 'Enabled' : 'Disabled'})
            </span>
          </div>
          
          {server.type === 'stdio' && (
            <>
              <div>Command: <span className="text-text">{server.config.command}</span></div>
              <div>Arguments: <span className="text-text">{Array.isArray(server.config.args) ? server.config.args.join(' ') : ''}</span></div>
              {server.config.env && Object.keys(server.config.env).length > 0 && (
                <div>
                  Environment Variables:
                  <div className="ml-2 text-text">
                    {Object.entries(server.config.env).map(([key, value]) => (
                      <div key={key}>{key}={value}</div>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}
          
          {server.type === 'sse' && (
            <>
              <div>URL: <span className="text-text">{server.config.url}</span></div>
              {server.config.headers && server.config.headers.length > 0 && (
                <div>
                  Headers:
                  <div className="ml-2 text-text">
                    {server.config.headers.map((header, idx) => (
                      <div key={idx}>{header}</div>
                    ))}
                  </div>
                </div>
              )}
            </>
          )}
        </div>
      </div>
    );
  };

  const renderServerForm = () => {
    return (
      <div className="mt-4 border border-border rounded-md p-3 space-y-3">
        <h4 className="font-medium text-sm">{isEditingServer ? 'Edit Server' : 'Add New Server'}</h4>
        
        <div className="space-y-3">
          <div className="flex items-center gap-4">
            <label className="text-xs text-muted-foreground w-20">Name</label>
            <input
              type="text"
              name="name"
              value={serverForm.name}
              onChange={handleServerFormChange}
              disabled={isEditingServer} // Can't change name when editing
              className="flex-1 text-sm bg-background rounded-md border border-border px-2 py-1"
              placeholder="server_name"
              autoComplete="off"
              autoCorrect="off"
              spellCheck="false"
            />
          </div>
          
          <div className="flex items-center gap-4">
            <label className="text-xs text-muted-foreground w-20">Type</label>
            <select
              name="type"
              value={serverForm.type}
              onChange={handleServerFormChange}
              disabled={isEditingServer} // Can't change type when editing
              className="flex-1 text-sm bg-background rounded-md border border-border px-2 py-1.5 appearance-none cursor-pointer"
            >
              <option value="stdio">STDIO</option>
              <option value="sse">SSE</option>
            </select>
          </div>
          
          <div className="flex items-center gap-4">
            <label className="text-xs text-muted-foreground w-20">Enabled</label>
            <input
              type="checkbox"
              name="enabled"
              checked={serverForm.enabled}
              onChange={handleServerFormChange}
              className="h-4 w-4 rounded border-border text-primary focus:ring-1 focus:ring-primary"
            />
            <span className="text-xs text-muted-foreground">
              {serverForm.enabled ? 'Yes' : 'No'}
            </span>
          </div>
          
          {serverForm.type === 'stdio' && (
            <>
              <div className="flex items-center gap-4">
                <label className="text-xs text-muted-foreground w-20">Command</label>
                <input
                  type="text"
                  name="command"
                  value={serverForm.command}
                  onChange={handleServerFormChange}
                  className="flex-1 text-sm bg-background rounded-md border border-border px-2 py-1"
                  placeholder="/usr/bin/node"
                  autoComplete="off"
                  autoCorrect="off"
                  spellCheck="false"
                />
              </div>
              
              <div className="flex items-center gap-4">
                <label className="text-xs text-muted-foreground w-20">Arguments</label>
                <input
                  type="text"
                  name="args"
                  value={serverForm.args}
                  onChange={handleServerFormChange}
                  className="flex-1 text-sm bg-background rounded-md border border-border px-2 py-1"
                  placeholder="index.js --port 8080"
                  autoComplete="off"
                  autoCorrect="off"
                  spellCheck="false"
                />
              </div>
              
              <div className="flex items-start gap-4">
                <label className="text-xs text-muted-foreground w-20 pt-1">Environment</label>
                <textarea
                  name="env"
                  value={serverForm.env}
                  onChange={handleServerFormChange}
                  rows={3}
                  className="flex-1 text-sm bg-background rounded-md border border-border px-2 py-1"
                  placeholder="DEBUG=true
PORT=8080"
                  autoComplete="off"
                  autoCorrect="off"
                  spellCheck="false"
                />
              </div>
            </>
          )}
          
          {serverForm.type === 'sse' && (
            <>
              <div className="flex items-center gap-4">
                <label className="text-xs text-muted-foreground w-20">URL</label>
                <input
                  type="text"
                  name="url"
                  value={serverForm.url}
                  onChange={handleServerFormChange}
                  className="flex-1 text-sm bg-background rounded-md border border-border px-2 py-1"
                  placeholder="https://api.example.com/mcp"
                  autoComplete="off"
                  autoCorrect="off"
                  spellCheck="false"
                />
              </div>
              
              <div className="flex items-start gap-4">
                <label className="text-xs text-muted-foreground w-20 pt-1">Headers</label>
                <textarea
                  name="headers"
                  value={serverForm.headers}
                  onChange={handleServerFormChange}
                  rows={3}
                  className="flex-1 text-sm bg-background rounded-md border border-border px-2 py-1"
                  placeholder="Authorization: Bearer token123
Content-Type: application/json"
                  autoComplete="off"
                  autoCorrect="off"
                  spellCheck="false"
                />
              </div>
            </>
          )}
        </div>
        
        <div className="flex justify-end gap-2 pt-2">
          <button
            onClick={handleCancelServerAction}
            className="px-3 py-1 text-xs rounded-md bg-muted/50 text-muted-foreground hover:bg-muted transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSaveServer}
            disabled={mcpActionStatus.loading || !serverForm.name || (serverForm.type === 'stdio' && !serverForm.command) || (serverForm.type === 'sse' && !serverForm.url)}
            className="px-3 py-1 text-xs rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {mcpActionStatus.loading ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>
    );
  };

  if (isLoading) {
    return <div className="p-4 text-center">Loading MCP settings...</div>;
  }

  return (
    <section className="space-y-3 pt-4">
      <div className="flex justify-between items-center">
        <h3 className="text-sm font-medium">MCP Servers</h3>
        <button 
          onClick={handleAddServer}
          disabled={isAddingServer || isEditingServer}
          className="flex items-center gap-1.5 px-3 py-1 text-xs rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <PlusCircle size={14} />
          Add Server
        </button>
      </div>
      
      {mcpActionStatus.success && (
        <div className="text-sm text-green-500 animate-fade-in">
          Operation completed successfully!
        </div>
      )}
      
      {mcpActionStatus.error && (
        <div className="text-sm text-red-500">
          {mcpActionStatus.error}
        </div>
      )}
      
      {(isAddingServer || isEditingServer) ? (
        renderServerForm()
      ) : (
        <>
          {mcpServers.length > 0 ? (
            <div className="space-y-2">
              <select
                value={selectedServer ? selectedServer.name : ''}
                onChange={(e) => {
                  const selected = mcpServers.find(s => s.name === e.target.value);
                  setSelectedServer(selected || null);
                }}
                className="w-full text-sm bg-background rounded-md border border-border px-2 py-1.5 appearance-none cursor-pointer"
              >
                <option value="">Select a server</option>
                {mcpServers.map(server => (
                  <option key={server.name} value={server.name}>
                    {server.name} ({server.type}) {server.isActive ? '- Active' : ''}
                  </option>
                ))}
              </select>
              
              {selectedServer && renderServerDetails(selectedServer)}
            </div>
          ) : (
            <div className="text-center py-6 text-sm text-muted-foreground">
              No MCP servers configured. Click "Add Server" to create one.
            </div>
          )}
        </>
      )}
    </section>
  );
}

export default MCPSettingsComponent;