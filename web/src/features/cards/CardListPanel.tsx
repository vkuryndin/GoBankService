import { Button } from '../../components/ui/Button'
import type { CardResponse } from '../../types/card'
import { getCardBadgeClass, getCardDisplayNumber, getCardStatusText } from '../../utils/format'

type CardListPanelProps = {
  cards: CardResponse[]
  selectedCardId: string
  onSelect: (card: CardResponse) => void
}

export function CardListPanel({ cards, selectedCardId, onSelect }: CardListPanelProps) {
  return (
    <section className="subPanel">
      <div className="subPanelHeader">
        <h3>Мои карты</h3>
        <span>{cards.length}</span>
      </div>

      {cards.length === 0 && (
        <div className="empty">Список пуст. Нажми “Загрузить карты” или выпусти новую карту.</div>
      )}

      {cards.length > 0 && (
        <div className="cardList">
          {cards.map((card) => (
            <Button
              key={card.id}
              className={selectedCardId === String(card.id) ? 'bankCardItem active' : 'bankCardItem'}
              type="button"
              onClick={() => onSelect(card)}
              aria-pressed={selectedCardId === String(card.id)}
            >
              <span className="cardNumber">{getCardDisplayNumber(card)}</span>
              <span className="cardMeta">
                <span>ID {card.id}</span>
                <span>account_id {card.account_id}</span>
                <span className={getCardBadgeClass(card)}>{getCardStatusText(card)}</span>
              </span>
            </Button>
          ))}
        </div>
      )}
    </section>
  )
}
