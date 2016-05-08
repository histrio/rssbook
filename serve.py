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


@route('/static/<filename>')
def server_static(filename):
    return static_file(filename, root='/path/to/your/static/files')

run(host='localhost', port=8080)
