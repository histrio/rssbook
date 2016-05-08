from bottle import route, run
from bottle import static_file
from bottle import get, post


@get('/login')
def login():
    return '''
        <a href=#> Login </a>
    '''


@post('/login')
def do_login():
    return "<p>Login failed.</p>"


@route('/assets/<filename:path>')
def server_static(filename):
    return static_file(filename, root='./static/assets/')


@route('/')
def static_index():
    return static_file("index.html", root='static')


run(host='localhost', port=8080)
