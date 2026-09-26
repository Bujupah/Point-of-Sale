import { useEffect, useState } from 'react'
import { api } from '../../services/api'
import { useI18n } from '../../i18n'
import type { Category } from '../../types'

interface Props {
  activeId: number | 'all' | 'favorites'
  onChange: (id: number | 'all' | 'favorites') => void
}

export function CategoryBar({ activeId, onChange }: Props) {
  const { t } = useI18n()
  const [categories, setCategories] = useState<Category[]>([])

  useEffect(() => {
    api.get<Category[]>('/api/categories').then((r) => setCategories(r ?? [])).catch(() => {})
  }, [])

  return (
    <div className="category-bar">
      <button className={`chip ${activeId === 'all' ? 'chip-active' : ''}`} onClick={() => onChange('all')}>
        {t('category_all')}
      </button>
      <button className={`chip ${activeId === 'favorites' ? 'chip-active' : ''}`} onClick={() => onChange('favorites')}>
        ⭐ {t('category_favorites')}
      </button>
      {categories.map((c) => (
        <button key={c.id} className={`chip ${activeId === c.id ? 'chip-active' : ''}`} onClick={() => onChange(c.id)}>
          {c.name}
        </button>
      ))}
    </div>
  )
}
