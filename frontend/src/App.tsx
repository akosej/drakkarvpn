import { useState, useEffect, useRef } from 'react';
import './App.css';
import bg from './assets/images/bg.png';
import { GetProfiles, SaveProfile, DeleteProfile, Connect, Disconnect, IsRunning, GetMetrics } from "../wailsjs/go/main/App";
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, AreaChart, Area } from 'recharts';

interface Profile {
  id: string;
  name: string;
  privateKey: string;
  publicKey: string;
  address: string;
  dns: string;
  endpoint: string;
  allowedIPs: string;
}

interface MetricPoint {
  time: string;
  tx: number;
  rx: number;
}

function App() {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [running, setRunning] = useState(false);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState<Profile>({
    id: '', name: '', privateKey: '', publicKey: '', address: '10.0.0.2/24', dns: '8.8.8.8', endpoint: '', allowedIPs: '0.0.0.0/0'
  });
  const [chartData, setChartData] = useState<MetricPoint[]>([]);
  const lastMetrics = useRef({ tx: 0, rx: 0 });

  useEffect(() => {
    loadData();
    const interval = setInterval(async () => {
      const isRunning = await IsRunning();
      setRunning(isRunning);
      
      const metrics = await GetMetrics();
      const newTx = metrics.tx - lastMetrics.current.tx;
      const newRx = metrics.rx - lastMetrics.current.rx;
      lastMetrics.current = { tx: metrics.tx, rx: metrics.rx };

      setChartData(prev => {
        const newData = [...prev, { 
          time: new Date().toLocaleTimeString().split(' ')[0], 
          tx: Math.max(0, newTx), 
          rx: Math.max(0, newRx) 
        }];
        return newData.slice(-30); 
      });
    }, 1000);
    return () => clearInterval(interval);
  }, []);

  async function loadData() {
    const p = await GetProfiles();
    setProfiles(p || []);
    const isRunning = await IsRunning();
    setRunning(isRunning);
    if (!selectedId && p && p.length > 0) {
        setSelectedId(p[0].id);
    }
  }

  const handleSave = async () => {
    await SaveProfile(formData as any);
    setShowForm(false);
    loadData();
  };

  const handleConnect = async (id: string) => {
    try {
      await Connect(id);
      setRunning(true);
    } catch (e) {
      alert(e);
    }
  };

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  return (
    <div className="layout">
      <div className="sidebar">
        <div className="sidebar-header">
          <h2></h2>
          <button onClick={() => { 
            setShowForm(true); 
            setFormData({id: '', name: '', privateKey: '', publicKey: '', address: '10.0.0.2/24', dns: '8.8.8.8', endpoint: '', allowedIPs: '0.0.0.0/0'}); 
          }}>+ Nuevo Perfil</button>
        </div>
        <div className="profile-list">
          {profiles.map(p => (
            <div 
              key={p.id} 
              className={`profile-item ${selectedId === p.id ? 'active' : ''}`}
              onClick={() => { setSelectedId(p.id); setShowForm(false); }}
            >
              <div className="profile-info">
                <strong>{p.name}</strong>
                <small>{p.endpoint}</small>
              </div>
              <div className="profile-actions">
                {!running && <button className="btn-connect" onClick={(e) => { e.stopPropagation(); handleConnect(p.id); }}>ON</button>}
                {running && selectedId === p.id && <button className="btn-disconnect" onClick={(e) => { e.stopPropagation(); Disconnect(); }}>OFF</button>}
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="main-content">
        {showForm ? (
          <div className="form-container">
            <h3>{formData.id ? 'Editar' : 'Nuevo'} Perfil</h3>
            <div className="grid-form">
              <div className="input-group">
                <label>Nombre</label>
                <input placeholder="Mi Conexión" value={formData.name} onChange={e => setFormData({...formData, name: e.target.value})} />
              </div>
              <div className="input-group">
                <label>WebSocket URL</label>
                <input placeholder="wss://mi-servidor.com/wg" value={formData.endpoint} onChange={e => setFormData({...formData, endpoint: e.target.value})} />
              </div>
              <div className="input-group">
                <label>IP VPN (Address)</label>
                <input placeholder="10.0.0.2/24" value={formData.address} onChange={e => setFormData({...formData, address: e.target.value})} />
              </div>
              <div className="input-group">
                <label>Servidores DNS</label>
                <input placeholder="8.8.8.8, 1.1.1.1" value={formData.dns} onChange={e => setFormData({...formData, dns: e.target.value})} />
              </div>
              <div className="input-group">
                <label>Allowed IPs</label>
                <input placeholder="0.0.0.0/0" value={formData.allowedIPs} onChange={e => setFormData({...formData, allowedIPs: e.target.value})} />
              </div>
              <div className="input-group">
                <label>Private Key (Cliente)</label>
                <input placeholder="Clave privada..." value={formData.privateKey} onChange={e => setFormData({...formData, privateKey: e.target.value})} />
              </div>
              <div className="input-group" style={{ gridColumn: 'span 2' }}>
                <label>Public Key (Servidor)</label>
                <input placeholder="Clave pública del servidor..." value={formData.publicKey} onChange={e => setFormData({...formData, publicKey: e.target.value})} />
              </div>
            </div>
            <div className="form-btns">
              <button className="btn-save" onClick={handleSave}>Guardar</button>
              <button className="btn-cancel" onClick={() => setShowForm(false)}>Cancelar</button>
            </div>
          </div>
        ) : selectedId ? (
          <div className="dashboard">
            <div className="dash-header">
              <h1>{profiles.find(p => p.id === selectedId)?.name}</h1>
              <div className={`status-pill ${running ? 'online' : 'offline'}`}>
                {running ? 'CONECTADO' : 'DESCONECTADO'}
              </div>
            </div>

            <div className="metrics-grid">
              <div className="metric-card">
                <span className="label">ENVIADO (TX)</span>
                <span className="value">{formatBytes(lastMetrics.current.tx)}</span>
              </div>
              <div className="metric-card">
                <span className="label">RECIBIDO (RX)</span>
                <span className="value">{formatBytes(lastMetrics.current.rx)}</span>
              </div>
            </div>

            <div className="chart-container">
              <h3>Tráfico en tiempo real</h3>
              <div style={{ width: '100%', height: 300 }}>
                <ResponsiveContainer>
                  <AreaChart data={chartData}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#444" />
                    <XAxis dataKey="time" stroke="#888" />
                    <YAxis stroke="#888" />
                    <Tooltip contentStyle={{ backgroundColor: '#222', border: 'none', color: '#fff' }} />
                    <Area type="monotone" dataKey="tx" stroke="#28a745" fill="#28a745" fillOpacity={0.3} name="Subida (B/s)" />
                    <Area type="monotone" dataKey="rx" stroke="#007bff" fill="#007bff" fillOpacity={0.3} name="Bajada (B/s)" />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            </div>
            
            <div className="details">
              <button className="btn-edit" onClick={() => { setFormData(profiles.find(p => p.id === selectedId)!); setShowForm(true); }}>Editar Parfil</button>
              <button className="btn-delete" onClick={() => { if(confirm('¿Seguro que quieres eliminar este perfil?')) { DeleteProfile(selectedId); setSelectedId(null); loadData(); } }}>Eliminar Perfil</button>
            </div>
          </div>
        ) : (
          <div className="empty-state">
            Selecciona un perfil o crea uno nuevo para empezar.
          </div>
        )}
      </div>
    </div>
  );
}

export default App;
