export function isRequired(value: string): boolean {
  return value.trim().length > 0
}

export function validateRequired(value: string, fieldName: string): string {
  return isRequired(value) ? '' : `${fieldName} обязателен.`
}

export function validateEmail(value: string): string {
  const email = value.trim()
  if (!email) {
    return 'Email обязателен.'
  }

  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email) ? '' : 'Некорректный email.'
}

export function validatePassword(value: string): string {
  if (!value) {
    return 'Password обязателен.'
  }

  return value.length >= 8 ? '' : 'Password должен быть не короче 8 символов.'
}

export function validatePasswordConfirmation(password: string, confirmation: string): string {
  if (!confirmation) {
    return 'Повторный пароль обязателен.'
  }

  return password === confirmation ? '' : 'Пароли не совпадают.'
}

export function validateAmount(value: string): string {
  const amount = value.trim()
  if (!amount) {
    return 'Amount обязателен.'
  }

  return /^\d+(\.\d{1,2})?$/.test(amount) ? '' : 'Amount должен быть числом с максимум 2 знаками после точки.'
}

export function validatePositiveInteger(value: string, fieldName: string): string {
  const number = Number(value)
  return Number.isInteger(number) && number > 0 ? '' : `${fieldName} должен быть положительным целым числом.`
}

export function validateDays(value: string): string {
  const number = Number(value)
  return Number.isInteger(number) && number >= 1 && number <= 365
    ? ''
    : 'Days должен быть от 1 до 365.'
}

export function firstValidationError(...errors: string[]): string {
  return errors.find(Boolean) || ''
}
