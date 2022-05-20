import math

class SimpleMap():
    

    def __init__(self, w, h):

        # map
        self.w = w
        self.h = h
        self.sMap = [[b' ' for _ in range(self.w)] for _ in range(self.h)]

        # metadata 
        self._locations = 0
        self._lines = 0
        self._lines_rendered = 0
        

    def SetLocation(self, x, y, loc):

        self.sMap[y][x] = loc
        self._locations += 1
        self.lines = math.floor(self._locations / self.w)

        self._render()

        
    def _render(self):

        lines_to_render = math.floor(self._locations / self.w) - self._lines_rendered

        if lines_to_render < 1:
            return

        for l in range(lines_to_render):
            line = "".join(self.sMap[self.h - 1 - self._lines_rendered - l][i].decode() for i in range(self.w))
            print(line)
            self._lines_rendered += 1
