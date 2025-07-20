# Установка плагина

## Системные требования

- Mattermost версии 6.0 или выше
- Права администратора для установки плагина
- Docker (для локального запуска Ollama)
- Доступ к интернету (для OpenAI API)

## Способы установки

### Установка через System Console

1. **Скачайте плагин**
   - Перейдите на [страницу релизов](https://github.com/EgorTarasov/summary/releases)
   - Скачайте файл `com.github.egortarasov.summary-X.X.X.tar.gz`

2. **Установите плагин**
   - Войдите в Mattermost как администратор
   - Перейдите в System Console → Plugins → Management
   - Нажмите "Choose File" и выберите скачанный файл
   - Нажмите "Upload"

3. **Активируйте плагин**
   - Найдите плагин "Summary" в списке
   - Нажмите "Enable" для активации

### Установка через командную строку

Если у вас есть доступ к серверу Mattermost:

```bash
# Скачайте плагин
wget https://github.com/EgorTarasov/summary/releases/latest/download/com.github.egortarasov.summary.tar.gz

# Установите плагин
sudo -u mattermost /opt/mattermost/bin/mattermost plugin add com.github.egortarasov.summary.tar.gz

# Активируйте плагин
sudo -u mattermost /opt/mattermost/bin/mattermost plugin enable com.github.egortarasov.summary
```

## Настройка LLM провайдера

После установки плагина необходимо настроить провайдер для работы с моделями машинного обучения.

### Вариант 1: Ollama (локальные модели)

1. **Установите Ollama**
   ```bash
   curl -fsSL https://ollama.ai/install.sh | sh
   ```

2. **Запустите модель**
   ```bash
   ollama pull llama3.2
   ollama serve
   ```

3. **Настройте плагин**
   - В System Console → Plugins → Summary
   - Установите `LLM Provider` в `ollama`
   - Укажите URL: `http://localhost:11434`
   - Выберите модель: `llama3.2`

### Вариант 2: OpenAI API

1. **Получите API ключ**
   - Зарегистрируйтесь на [platform.openai.com](https://platform.openai.com)
   - Создайте API ключ в разделе API Keys

2. **Настройте плагин**
   - В System Console → Plugins → Summary
   - Установите `LLM Provider` в `openai`
   - Введите API ключ
   - Выберите модель: `gpt-4` или `gpt-3.5-turbo`

## Проверка установки

1. Перейдите в любой канал
2. Создайте тред с несколькими сообщениями
3. В треде выполните команду `/summary`
4. Если плагин работает корректно, вы получите резюме сообщений

## Устранение неполадок

### Плагин не активируется

!!! warning "Ошибка активации"
    Если плагин не активируется, проверьте:
    - Версию Mattermost (должна быть 6.0+)
    - Логи сервера в System Console → Reporting → Server Logs
    - Права доступа к файлам плагина

### LLM не отвечает

!!! error "Нет ответа от модели"
    Возможные причины:
    - Неверная настройка провайдера
    - Недоступность сервиса (Ollama не запущен, нет интернета для OpenAI)
    - Неверный API ключ
    - Исчерпание лимитов API

### Команда /summary не работает

!!! info "Команда недоступна"
    Убедитесь, что:
    - Плагин активирован
    - У вас есть права на выполнение slash-команд
    - Вы находитесь в треде с сообщениями
