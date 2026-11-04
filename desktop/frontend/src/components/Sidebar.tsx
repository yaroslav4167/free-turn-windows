import {backend} from '../api'
import {tr} from '../language'

interface Props {
  servers: backend.Server[]
  selectedId: string | null
  activeState: string
  activeServerId: string
  onSelect: (id: string) => void
  onAdd: () => void
}

export default function Sidebar({servers, selectedId, activeState, activeServerId, onSelect, onAdd}: Props) {
  return (
    <div className="sidebar">
      <div className="sidebar-header">
        <h1>{tr('Серверы', 'Servers')}</h1>
        <span className="version">{tr('Профили подключения', 'Connection profiles')} · {servers.length}</span>
      </div>
      <div className="server-list">
        {servers.map((s) => {
          const isActive = s.id === activeServerId && activeState !== 'idle'
          return (
            <button
              type="button"
              key={s.id}
              className={'server-item' + (s.id === selectedId ? ' active' : '')}
              onClick={() => onSelect(s.id)}
            >
              <span className={'dot' + (isActive ? ' ' + activeState : '')} />
              <span className="name">
                {s.name || s.peer || tr('Без имени', 'Unnamed')}
                <span className="peer">{s.peer}</span>
              </span>
            </button>
          )
        })}
        {servers.length === 0 && (
          <div style={{color: 'var(--text-dim)', padding: '10px', fontSize: '12px'}}>{tr('Серверов пока нет', 'No servers yet')}</div>
        )}
      </div>
      <div className="sidebar-footer">
        <button className="btn-add" onClick={onAdd}>
          + {tr('Добавить сервер', 'Add server')}
        </button>
      </div>
    </div>
  )
}
