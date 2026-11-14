package voicechat

type topicSeed struct {
	topic    string
	greeting string
}

var topicSeeds = map[string][]topicSeed{
	"en": {
		{"Favorite hobby", "Hi! I'm glad you're here. Let's talk about hobbies today. What do you enjoy doing in your free time?"},
		{"Travel dreams", "Hello and welcome! Today's topic is travel. If you could go anywhere in the world, where would you go and why?"},
		{"Food you love", "Hi there! Let's chat about food today. What's your favorite dish, and how often do you eat it?"},
		{"A book or film you enjoyed", "Welcome! Today we'll talk about books and movies. Can you tell me about one you really enjoyed?"},
		{"Your daily routine", "Hello! Let's talk about everyday life. Can you describe a typical day for you?"},
		{"A memorable day", "Hi! Today's topic is memories. Tell me about a day you will never forget."},
		{"Your hometown", "Welcome! Let's talk about where you're from. What is your hometown like?"},
		{"Future plans", "Hi! Today we'll talk about the future. What's something you're looking forward to?"},
		{"Music you like", "Hello! Let's chat about music. What kind of music do you enjoy, and who is your favorite artist?"},
		{"A person you admire", "Hi there! Today we'll talk about people who inspire us. Who is someone you admire, and why?"},
	},
	"es": {
		{"Tu pasatiempo favorito", "¡Hola! Me alegro de que estés aquí. Hoy vamos a hablar de pasatiempos. ¿Qué te gusta hacer en tu tiempo libre?"},
		{"Viajes soñados", "¡Hola y bienvenido! Hoy hablaremos de viajes. Si pudieras ir a cualquier lugar del mundo, ¿a dónde irías y por qué?"},
		{"Tu comida favorita", "¡Hola! Vamos a hablar de comida. ¿Cuál es tu plato favorito y con qué frecuencia lo comes?"},
		{"Un libro o una película", "¡Bienvenido! Hoy hablaremos de libros y películas. ¿Puedes contarme sobre uno que te haya gustado mucho?"},
		{"Tu rutina diaria", "¡Hola! Hablemos de la vida cotidiana. ¿Cómo es un día normal para ti?"},
		{"Un día memorable", "¡Hola! Hoy hablaremos de recuerdos. Cuéntame de un día que nunca olvidarás."},
		{"Tu ciudad natal", "¡Bienvenido! Hablemos de tu lugar de origen. ¿Cómo es tu ciudad natal?"},
		{"Planes para el futuro", "¡Hola! Hoy hablaremos del futuro. ¿Qué es algo que esperas con ganas?"},
		{"La música que te gusta", "¡Hola! Vamos a hablar de música. ¿Qué tipo de música te gusta y quién es tu artista favorito?"},
		{"Una persona que admiras", "¡Hola! Hoy hablaremos de personas que nos inspiran. ¿A quién admiras y por qué?"},
	},
	"fr": {
		{"Ton passe-temps préféré", "Bonjour ! Je suis content de te voir. Aujourd'hui, parlons des loisirs. Qu'est-ce que tu aimes faire pendant ton temps libre ?"},
		{"Voyages de rêve", "Bonjour et bienvenue ! Aujourd'hui, parlons de voyages. Si tu pouvais aller n'importe où dans le monde, où irais-tu et pourquoi ?"},
		{"Ton plat préféré", "Bonjour ! Parlons de nourriture. Quel est ton plat préféré et à quelle fréquence le manges-tu ?"},
		{"Un livre ou un film", "Bienvenue ! Aujourd'hui, parlons de livres et de films. Peux-tu me parler d'un livre ou d'un film que tu as beaucoup aimé ?"},
		{"Ta routine quotidienne", "Bonjour ! Parlons de la vie de tous les jours. À quoi ressemble une journée typique pour toi ?"},
		{"Une journée mémorable", "Bonjour ! Aujourd'hui, parlons de souvenirs. Raconte-moi une journée que tu n'oublieras jamais."},
		{"Ta ville natale", "Bienvenue ! Parlons de l'endroit d'où tu viens. Comment est ta ville natale ?"},
		{"Tes projets d'avenir", "Bonjour ! Aujourd'hui, parlons de l'avenir. Qu'est-ce que tu attends avec impatience ?"},
		{"La musique que tu aimes", "Bonjour ! Parlons de musique. Quel genre de musique aimes-tu et qui est ton artiste préféré ?"},
		{"Une personne que tu admires", "Bonjour ! Aujourd'hui, parlons des gens qui nous inspirent. Qui admires-tu et pourquoi ?"},
	},
}
